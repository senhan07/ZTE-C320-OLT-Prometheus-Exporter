package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metric descriptions for the ZTE OLT exporter.
var (
	// OnuStatusGaugeDesc describes the operational status of the ONU.
	OnuStatusGaugeDesc = prometheus.NewDesc(
		"zte_onu_status",
		"The operational status of the ONU (1=Online, 2=DyingGasp, 3=LOS, 4=PowerOff, 0=Other).",
		[]string{"serial_number"}, nil,
	)

	// OnuMappingInfoGaugeDesc provides a mapping of serial numbers to descriptive labels.
	OnuMappingInfoGaugeDesc = prometheus.NewDesc(
		"zte_onu_mapping_info",
		"Information mapping for the ZTE ONU device.",
		[]string{"board", "pon", "onu_id", "name", "serial_number", "onu_type", "description", "offline_reason", "ip_address"}, nil,
	)

	// OnuRxPowerGaugeDesc describes the received optical power of the ONU.
	OnuRxPowerGaugeDesc = prometheus.NewDesc(
		"zte_onu_rx_power_dbm",
		"The received optical power of the ONU in dBm.",
		[]string{"serial_number"}, nil,
	)

	// OnuTxPowerGaugeDesc describes the transmitted optical power of the ONU.
	OnuTxPowerGaugeDesc = prometheus.NewDesc(
		"zte_onu_tx_power_dbm",
		"The transmitted optical power of the ONU in dBm.",
		[]string{"serial_number"}, nil,
	)

	// OnuUptimeGaugeDesc describes the uptime of the ONU in seconds.
	OnuUptimeGaugeDesc = prometheus.NewDesc(
		"zte_onu_uptime_seconds",
		"The uptime of the ONU in seconds.",
		[]string{"serial_number"}, nil,
	)

	// OnuLastDownDurationGaugeDesc describes the duration of the last downtime in seconds.
	OnuLastDownDurationGaugeDesc = prometheus.NewDesc(
		"zte_onu_last_down_duration_seconds",
		"The duration of the last downtime in seconds.",
		[]string{"serial_number"}, nil,
	)

	// OnuLastOnlineGaugeDesc describes the last online timestamp as a Unix epoch.
	OnuLastOnlineGaugeDesc = prometheus.NewDesc(
		"zte_onu_last_online_timestamp_seconds",
		"The last online timestamp of the ONU as a Unix epoch.",
		[]string{"serial_number"}, nil,
	)

	// OnuLastOfflineGaugeDesc describes the last offline timestamp as a Unix epoch.
	OnuLastOfflineGaugeDesc = prometheus.NewDesc(
		"zte_onu_last_offline_timestamp_seconds",
		"The last offline timestamp of the ONU as a Unix epoch.",
		[]string{"serial_number"}, nil,
	)

	// OnuGponOpticalDistanceGaugeDesc describes the GPON optical distance in meters.
	OnuGponOpticalDistanceGaugeDesc = prometheus.NewDesc(
		"zte_onu_gpon_optical_distance_meters",
		"The GPON optical distance to the ONU in meters.",
		[]string{"serial_number"}, nil,
	)
)