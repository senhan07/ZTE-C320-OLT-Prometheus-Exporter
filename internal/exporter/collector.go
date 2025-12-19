package exporter

import (
	"context"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/megadata-dev/go-snmp-olt-zte-c320/internal/model"
	"github.com/megadata-dev/go-snmp-olt-zte-c320/internal/usecase"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// OnuCollector implements the prometheus.Collector interface.
type OnuCollector struct {
	onuUsecase    usecase.OnuUseCaseInterface
	boardMin      int
	boardMax      int
	ponMin        int
	ponMax        int
	maxConcurrent int
}

// NewOnuCollector creates a new OnuCollector and configures the scan range.
func NewOnuCollector(onuUsecase usecase.OnuUseCaseInterface) *OnuCollector {
	// Get scan range from environment variables or use defaults.
	boardMin, _ := strconv.Atoi(os.Getenv("PROMETHEUS_BOARD_MIN"))
	boardMax, _ := strconv.Atoi(os.Getenv("PROMETHEUS_BOARD_MAX"))
	ponMin, _ := strconv.Atoi(os.Getenv("PROMETHEUS_PON_MIN"))
	ponMax, _ := strconv.Atoi(os.Getenv("PROMETHEUS_PON_MAX"))
	maxConcurrent, _ := strconv.Atoi(os.Getenv("PROMETHEUS_MAX_CONCURRENT"))

	// Set default values if not provided.
	if boardMin == 0 {
		boardMin = 1
	}
	if boardMax == 0 {
		boardMax = 2
	}
	if ponMin == 0 {
		ponMin = 1
	}
	if ponMax == 0 {
		ponMax = 16
	}
	if maxConcurrent == 0 {
		maxConcurrent = 10 // Default to 10 concurrent scrapers
	}

	return &OnuCollector{
		onuUsecase:    onuUsecase,
		boardMin:      boardMin,
		boardMax:      boardMax,
		ponMin:        ponMin,
		ponMax:        ponMax,
		maxConcurrent: maxConcurrent,
	}
}

// Describe sends the static descriptions of all metrics collected by the exporter.
func (c *OnuCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- OnuStatusGaugeDesc
	ch <- OnuMappingInfoGaugeDesc
	ch <- OnuRxPowerGaugeDesc
	ch <- OnuTxPowerGaugeDesc
	ch <- OnuUptimeGaugeDesc
	ch <- OnuLastDownDurationGaugeDesc
	ch <- OnuLastOnlineGaugeDesc
	ch <- OnuLastOfflineGaugeDesc
	ch <- OnuGponOpticalDistanceGaugeDesc
}

// Collect fetches the metrics from the OLT and delivers them to Prometheus.
func (c *OnuCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Scrape timeout
	defer cancel()

	log.Info().Msg("Starting metric collection for Prometheus scrape")
	startTime := time.Now()

	var wg sync.WaitGroup
	uniqueOnus := sync.Map{} // Use a concurrent-safe map for unique ONUs.
	jobs := make(chan model.ONUInfoPerBoard, 1000)

	// Start the main worker pool.
	for i := 0; i < c.maxConcurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for discoveredOnu := range jobs {
				detailedOnu, err := c.onuUsecase.GetByBoardIDPonIDAndOnuID(
					discoveredOnu.Board, discoveredOnu.PON, discoveredOnu.ID,
				)
				if err != nil {
					log.Warn().Err(err).Int("board", discoveredOnu.Board).Int("pon", discoveredOnu.PON).Int("onu_id", discoveredOnu.ID).Msg("Failed to get detailed ONU info")
					continue
				}

				if detailedOnu.SerialNumber == "" {
					continue // Cannot process ONUs without a serial number.
				}

				// Filter for unique serial numbers, prioritizing non-unknown statuses.
				if existing, exists := uniqueOnus.Load(detailedOnu.SerialNumber); exists {
					existingOnu := existing.(model.ONUCustomerInfo)
					if mapStatusToNumeric(existingOnu.Status) == 0 && mapStatusToNumeric(detailedOnu.Status) != 0 {
						uniqueOnus.Store(detailedOnu.SerialNumber, detailedOnu)
					}
				} else {
					uniqueOnus.Store(detailedOnu.SerialNumber, detailedOnu)
				}
			}
		}()
	}

	// Discover all ONUs and feed them into the jobs channel.
	for boardID := c.boardMin; boardID <= c.boardMax; boardID++ {
		for ponID := c.ponMin; ponID <= c.ponMax; ponID++ {
			discoveredOnus, err := c.onuUsecase.GetByBoardIDAndPonID(ctx, boardID, ponID)
			if err != nil {
				log.Warn().Err(err).Int("board", boardID).Int("pon", ponID).Msg("Failed to discover ONUs")
				continue
			}
			for _, onu := range discoveredOnus {
				jobs <- onu
			}
		}
	}
	close(jobs)

	// Wait for all workers to finish fetching and filtering.
	wg.Wait()

	// Send metrics for the unique ONUs.
	totalOnusProcessed := 0
	uniqueOnus.Range(func(key, value interface{}) bool {
		detailedOnu := value.(model.ONUCustomerInfo)
		sendMetrics(ch, detailedOnu)
		totalOnusProcessed++
		return true
	})

	duration := time.Since(startTime)
	log.Info().Int("processed_onus", totalOnusProcessed).Str("duration", duration.String()).Msg("Finished metric collection for scrape")
}

// sendMetrics creates and sends Prometheus metrics for a single ONU.
func sendMetrics(ch chan<- prometheus.Metric, detailedOnu model.ONUCustomerInfo) {
	// Set ONU Mapping Info
	ch <- prometheus.MustNewConstMetric(
		OnuMappingInfoGaugeDesc,
		prometheus.GaugeValue,
		1,
		strconv.Itoa(detailedOnu.Board),
		strconv.Itoa(detailedOnu.PON),
		strconv.Itoa(detailedOnu.ID),
		detailedOnu.Name,
		detailedOnu.SerialNumber,
		detailedOnu.OnuType,
		detailedOnu.Description,
		detailedOnu.LastOfflineReason,
		detailedOnu.IPAddress,
	)

	// Set ONU Status
	ch <- prometheus.MustNewConstMetric(
		OnuStatusGaugeDesc,
		prometheus.GaugeValue,
		mapStatusToNumeric(detailedOnu.Status),
		detailedOnu.SerialNumber,
	)

	// Set power metrics only if the device is Online.
	if detailedOnu.Status == "Online" {
		if rxPower, err := strconv.ParseFloat(detailedOnu.RXPower, 64); err == nil {
			if rxPower < 100 { // Filter out invalid readings
				ch <- prometheus.MustNewConstMetric(OnuRxPowerGaugeDesc, prometheus.GaugeValue, rxPower, detailedOnu.SerialNumber)
				log.Debug().Str("serial_number", detailedOnu.SerialNumber).Float64("rx_power", rxPower).Msg("Successfully parsed and set RxPower")
			}
		} else {
			log.Warn().Err(err).Str("serial_number", detailedOnu.SerialNumber).Str("rx_power_str", detailedOnu.RXPower).Msg("Could not parse RxPower")
		}

		if txPower, err := strconv.ParseFloat(detailedOnu.TXPower, 64); err == nil {
			if txPower < 100 { // Filter out invalid readings
				ch <- prometheus.MustNewConstMetric(OnuTxPowerGaugeDesc, prometheus.GaugeValue, txPower, detailedOnu.SerialNumber)
			}
		} else {
			log.Warn().Err(err).Str("serial_number", detailedOnu.SerialNumber).Str("tx_power_str", detailedOnu.TXPower).Msg("Could not parse TxPower")
		}
	}

	// Set other metrics
	ch <- prometheus.MustNewConstMetric(OnuUptimeGaugeDesc, prometheus.GaugeValue, parseDurationStringToSeconds(detailedOnu.Uptime), detailedOnu.SerialNumber)
	ch <- prometheus.MustNewConstMetric(OnuLastDownDurationGaugeDesc, prometheus.GaugeValue, parseDurationStringToSeconds(detailedOnu.LastDownTimeDuration), detailedOnu.SerialNumber)
	ch <- prometheus.MustNewConstMetric(OnuLastOnlineGaugeDesc, prometheus.GaugeValue, parseTimestampStringToEpoch(detailedOnu.LastOnline), detailedOnu.SerialNumber)
	ch <- prometheus.MustNewConstMetric(OnuLastOfflineGaugeDesc, prometheus.GaugeValue, parseTimestampStringToEpoch(detailedOnu.LastOffline), detailedOnu.SerialNumber)
	if distance, err := strconv.ParseFloat(detailedOnu.GponOpticalDistance, 64); err == nil {
		ch <- prometheus.MustNewConstMetric(OnuGponOpticalDistanceGaugeDesc, prometheus.GaugeValue, distance, detailedOnu.SerialNumber)
	} else {
		log.Warn().Err(err).Str("serial_number", detailedOnu.SerialNumber).Str("distance_str", detailedOnu.GponOpticalDistance).Msg("Could not parse GponOpticalDistance")
	}
}

// --- Helper functions ---

// parseDurationStringToSeconds converts a duration string like "X days Y hours Z minutes W seconds" to total seconds.
func parseDurationStringToSeconds(durationStr string) float64 {
	var totalSeconds int64
	daysRegex := regexp.MustCompile(`(\d+)\s*days`)
	hoursRegex := regexp.MustCompile(`(\d+)\s*hours`)
	minutesRegex := regexp.MustCompile(`(\d+)\s*minutes`)
	secondsRegex := regexp.MustCompile(`(\d+)\s*seconds`)

	parseAndAddTime := func(regex *regexp.Regexp, multiplier int64) {
		if matches := regex.FindStringSubmatch(durationStr); len(matches) > 1 {
			value, err := strconv.ParseInt(matches[1], 10, 64)
			if err == nil {
				totalSeconds += value * multiplier
			}
		}
	}

	parseAndAddTime(daysRegex, 24*3600)
	parseAndAddTime(hoursRegex, 3600)
	parseAndAddTime(minutesRegex, 60)
	parseAndAddTime(secondsRegex, 1)

	return float64(totalSeconds)
}

// parseTimestampStringToEpoch converts a timestamp string (YYYY-MM-DD HH:MM:SS) to a Unix epoch.
func parseTimestampStringToEpoch(timestampStr string) float64 {
	timestampStr = strings.TrimSpace(timestampStr)
	if timestampStr == "" {
		return 0 // Return 0 for empty timestamps
	}
	layout := "2006-01-02 15:04:05"
	t, err := time.Parse(layout, timestampStr)
	if err != nil {
		log.Warn().Err(err).Str("timestamp", timestampStr).Msg("Could not parse timestamp string")
		return 0
	}
	return float64(t.Unix())
}

// mapStatusToNumeric maps the ONU status string to a numeric value.
func mapStatusToNumeric(status string) float64 {
	switch status {
	case "Online":
		return 1
	case "Dying Gasp":
		return 2
	case "LOS":
		return 3
	case "Power-Off":
		return 4
	default:
		return 0
	}
}