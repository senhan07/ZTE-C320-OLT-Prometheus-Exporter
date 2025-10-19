package exporter

import (
	"context"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/megadata-dev/go-snmp-olt-zte-c320/internal/usecase"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// OnuCollector is a struct that holds the use case for fetching ONU data.
type OnuCollector struct {
	onuUsecase usecase.OnuUseCaseInterface
}

// --- Helper functions for parsing ---

// parseDurationStringToSeconds converts a duration string like "X days Y hours Z minutes W seconds" to total seconds.
// It uses regular expressions to robustly parse the string.
func parseDurationStringToSeconds(durationStr string) float64 {
	var totalSeconds int64

	// Regular expressions for each time unit
	daysRegex := regexp.MustCompile(`(\d+)\s*days`)
	hoursRegex := regexp.MustCompile(`(\d+)\s*hours`)
	minutesRegex := regexp.MustCompile(`(\d+)\s*minutes`)
	secondsRegex := regexp.MustCompile(`(\d+)\s*seconds`)

	// Helper function to parse and add time from a regex match
	parseAndAddTime := func(regex *regexp.Regexp, multiplier int64) {
		if matches := regex.FindStringSubmatch(durationStr); len(matches) > 1 {
			value, err := strconv.ParseInt(matches[1], 10, 64)
			if err == nil {
				totalSeconds += value * multiplier
			}
		}
	}

	// Extract and sum all parts of the duration string
	parseAndAddTime(daysRegex, 24*3600)
	parseAndAddTime(hoursRegex, 3600)
	parseAndAddTime(minutesRegex, 60)
	parseAndAddTime(secondsRegex, 1)

	return float64(totalSeconds)
}

// parseTimestampStringToEpoch converts a timestamp string (YYYY-MM-DD HH:MM:SS) to a Unix epoch.
func parseTimestampStringToEpoch(timestampStr string) float64 {
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

// NewOnuCollector creates a new OnuCollector.
func NewOnuCollector(onuUsecase usecase.OnuUseCaseInterface) *OnuCollector {
	return &OnuCollector{onuUsecase: onuUsecase}
}

// Start runs the collector in a loop to periodically fetch data.
func (c *OnuCollector) Start(ctx context.Context) {
	// Get scan range from environment variables or use defaults.
	boardMin, _ := strconv.Atoi(os.Getenv("PROMETHEUS_BOARD_MIN"))
	boardMax, _ := strconv.Atoi(os.Getenv("PROMETHEUS_BOARD_MAX"))
	ponMin, _ := strconv.Atoi(os.Getenv("PROMETHEUS_PON_MIN"))
	ponMax, _ := strconv.Atoi(os.Getenv("PROMETHEUS_PON_MAX"))

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

	// Run the collection loop.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				log.Info().Msg("Starting ONU discovery and data collection for Prometheus")
				c.collect(ctx, boardMin, boardMax, ponMin, ponMax)
				log.Info().Msg("Finished ONU data collection")
				time.Sleep(30 * time.Second) // Collection interval
			}
		}
	}()
}

// collect performs a single run of the data collection.
func (c *OnuCollector) collect(ctx context.Context, boardMin, boardMax, ponMin, ponMax int) {
	// Reset gauges to remove old data to avoid reporting stale metrics.
	OnuMappingInfoGauge.Reset()
	OnuStatusGauge.Reset()
	OnuRxPowerGauge.Reset()
	OnuTxPowerGauge.Reset()
	OnuUptimeGauge.Reset()
	OnuLastDownDurationGauge.Reset()
	OnuLastOnlineGauge.Reset()
	OnuLastOfflineGauge.Reset()
	OnuGponOpticalDistanceGauge.Reset()

	for boardID := boardMin; boardID <= boardMax; boardID++ {
		for ponID := ponMin; ponID <= ponMax; ponID++ {
			// Discover active ONUs on the current board and PON.
			discoveredOnus, err := c.onuUsecase.GetByBoardIDAndPonID(ctx, boardID, ponID)
			if err != nil {
				log.Warn().Err(err).Int("board", boardID).Int("pon", ponID).Msg("Failed to discover ONUs")
				continue // Move to the next PON if discovery fails.
			}

			if len(discoveredOnus) == 0 {
				continue // No ONUs found, move to the next PON.
			}

			// Fetch detailed information for each discovered ONU.
			for _, discoveredOnu := range discoveredOnus {
				onuID := discoveredOnu.ID
				detailedOnu, err := c.onuUsecase.GetByBoardIDPonIDAndOnuID(boardID, ponID, onuID)
				if err != nil {
					log.Warn().Err(err).Int("board", boardID).Int("pon", ponID).Int("onu_id", onuID).Msg("Failed to get detailed ONU info")
					continue // Move to the next ONU.
				}

				// --- Update Prometheus Metrics ---

				labels := prometheus.Labels{
					"serial_number": detailedOnu.SerialNumber,
				}

				// Set ONU Mapping Info Gauge
				mappingLabels := prometheus.Labels{
					"board":          strconv.Itoa(detailedOnu.Board),
					"pon":            strconv.Itoa(detailedOnu.PON),
					"onu_id":         strconv.Itoa(detailedOnu.ID),
					"name":           detailedOnu.Name,
					"serial_number":  detailedOnu.SerialNumber,
					"onu_type":       detailedOnu.OnuType,
					"description":    detailedOnu.Description,
					"offline_reason": detailedOnu.LastOfflineReason,
					"ip_address":     detailedOnu.IPAddress,
				}
				OnuMappingInfoGauge.With(mappingLabels).Set(1)

				// Set ONU Status Gauge
				OnuStatusGauge.With(labels).Set(mapStatusToNumeric(detailedOnu.Status))

				// Only report power metrics if the device is Online.
				if detailedOnu.Status == "Online" {
					// Set ONU Rx Power Gauge
					if rxPower, err := strconv.ParseFloat(detailedOnu.RXPower, 64); err == nil {
						// Filter out invalid readings
						if rxPower < 100 {
							OnuRxPowerGauge.With(labels).Set(rxPower)
						}
					} else {
						log.Warn().Err(err).Msg("Could not parse RxPower")
					}

					// Set ONU Tx Power Gauge
					if txPower, err := strconv.ParseFloat(detailedOnu.TXPower, 64); err == nil {
						// Filter out invalid readings
						if txPower < 100 {
							OnuTxPowerGauge.With(labels).Set(txPower)
						}
					} else {
						log.Warn().Err(err).Msg("Could not parse TxPower")
					}
				}

				// Set other gauges
				OnuUptimeGauge.With(labels).Set(parseDurationStringToSeconds(detailedOnu.Uptime))
				OnuLastDownDurationGauge.With(labels).Set(parseDurationStringToSeconds(detailedOnu.LastDownTimeDuration))
				OnuLastOnlineGauge.With(labels).Set(parseTimestampStringToEpoch(detailedOnu.LastOnline))
				OnuLastOfflineGauge.With(labels).Set(parseTimestampStringToEpoch(detailedOnu.LastOffline))
				if distance, err := strconv.ParseFloat(detailedOnu.GponOpticalDistance, 64); err == nil {
					OnuGponOpticalDistanceGauge.With(labels).Set(distance)
				} else {
					log.Warn().Err(err).Msg("Could not parse GponOpticalDistance")
				}
			}
		}
	}
}