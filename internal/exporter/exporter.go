package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// OnuStatusGauge shows the operational status of the ONU.
	OnuStatusGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zte_onu_status",
			Help: "The operational status of the ONU (1=Online, 2=DyingGasp, 3=LOS, 4=PowerOff, 0=Other).",
		},
		[]string{"serial_number"},
	)

	// OnuMappingInfoGauge provides a mapping of serial numbers to descriptive labels.
	OnuMappingInfoGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zte_onu_mapping_info",
			Help: "Information mapping for the ZTE ONU device.",
		},
		[]string{"board", "pon", "onu_id", "name", "serial_number", "onu_type", "description", "offline_reason", "ip_address"},
	)

	// OnuRxPowerGauge shows the received optical power of the ONU.
	OnuRxPowerGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zte_onu_rx_power_dbm",
			Help: "The received optical power of the ONU in dBm.",
		},
		[]string{"serial_number"},
	)

	// OnuTxPowerGauge shows the transmitted optical power of the ONU.
	OnuTxPowerGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zte_onu_tx_power_dbm",
			Help: "The transmitted optical power of the ONU in dBm.",
		},
		[]string{"serial_number"},
	)

	// OnuUptimeGauge shows the uptime of the ONU in seconds.
	OnuUptimeGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zte_onu_uptime_seconds",
			Help: "The uptime of the ONU in seconds.",
		},
		[]string{"serial_number"},
	)

	// OnuLastDownDurationGauge shows the duration of the last downtime in seconds.
	OnuLastDownDurationGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zte_onu_last_down_duration_seconds",
			Help: "The duration of the last downtime in seconds.",
		},
		[]string{"serial_number"},
	)

	// OnuLastOnlineGauge shows the last online timestamp as a Unix epoch.
	OnuLastOnlineGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zte_onu_last_online_timestamp_seconds",
			Help: "The last online timestamp of the ONU as a Unix epoch.",
		},
		[]string{"serial_number"},
	)

	// OnuLastOfflineGauge shows the last offline timestamp as a Unix epoch.
	OnuLastOfflineGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zte_onu_last_offline_timestamp_seconds",
			Help: "The last offline timestamp of the ONU as a Unix epoch.",
		},
		[]string{"serial_number"},
	)

	// OnuGponOpticalDistanceGauge shows the GPON optical distance in meters.
	OnuGponOpticalDistanceGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zte_onu_gpon_optical_distance_meters",
			Help: "The GPON optical distance to the ONU in meters.",
		},
		[]string{"serial_number"},
	)
)