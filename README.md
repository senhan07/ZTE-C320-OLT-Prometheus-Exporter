# ZTE C320 OLT Prometheus Exporter

[![ci](https://github.com/Cepat-Kilat-Teknologi/go-snmp-olt-zte-c320/actions/workflows/ci.yml/badge.svg)](https://github.com/Cepat-Kilat-Teknologi/go-snmp-olt-zte-c320/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/megadata-dev/go-snmp-olt-zte-c320)](https://goreportcard.com/report/github.com/megadata-dev/go-snmp-olt-zte-c320)

This application is a Prometheus exporter for monitoring a ZTE C320 OLT using SNMP. It automatically discovers ONUs within a configurable range and exposes their metrics for collection by a Prometheus server.

## Getting Started

The easiest way to run the exporter is with Docker.

```shell
docker run -d -p 8081:8081 --name zte-olt-exporter \
-e SNMP_HOST=x.x.x.x \
-e SNMP_PORT=161 \
-e SNMP_COMMUNITY=your_community_string \
-e REDIS_HOST=your_redis_host \
-e REDIS_PORT=6379 \
sumitroajiprabowo/go-snmp-olt-zte-c320:latest
```

The exporter will be available on port `8081`.

## Configuration

The exporter is configured using environment variables.

| Variable                  | Description                               | Default | Required |
|---------------------------|-------------------------------------------|---------|----------|
| `SNMP_HOST`               | The IP address of the ZTE OLT.            |         | Yes      |
| `SNMP_PORT`               | The SNMP port of the OLT.                 | `161`   | No       |
| `SNMP_COMMUNITY`          | The SNMP community string for the OLT.    |         | Yes      |
| `REDIS_HOST`              | The hostname of the Redis server for caching. |         | Yes      |
| `REDIS_PORT`              | The port for the Redis server.            | `6379`  | No       |
| `REDIS_DB`                | The Redis database number to use.         | `0`     | No       |
| `REDIS_MIN_IDLE_CONNECTIONS`| The minimum number of idle connections to Redis. | `200`   | No       |
| `REDIS_POOL_SIZE`         | The Redis connection pool size.           | `12000` | No       |
| `REDIS_POOL_TIMEOUT`      | The Redis connection pool timeout.        | `240`   | No       |
| `PROMETHEUS_BOARD_MIN`    | The starting board number to scan for ONUs.| `1`     | No       |
| `PROMETHEUS_BOARD_MAX`    | The ending board number to scan for ONUs. | `2`     | No       |
| `PROMETHEUS_PON_MIN`      | The starting PON port number to scan.     | `1`     | No       |
| `PROMETHEUS_PON_MAX`      | The ending PON port number to scan.       | `16`    | No       |

## Prometheus Metrics

The exporter provides metrics on the `/metrics` endpoint. To ensure stable and reliable long-term monitoring, all numeric metrics (like power levels and uptime) are anchored to the ONU's `serial_number`. Descriptive labels that can change over time (like name, description, and physical location) are exposed in a separate `zte_onu_mapping_info` metric.

### Example Queries

**To get the Rx Power for all ONUs and show their names:**
```promql
zte_onu_rx_power_dbm * on(serial_number) group_left(name) zte_onu_mapping_info
```

**To count all devices that are not fully online:**
```promql
count(zte_onu_status != 1)
```

**To get a table of all devices that are not online, showing their name and specific status:**
```promql
zte_onu_status{serial_number!=""} * on(serial_number) group_left(name) zte_onu_mapping_info != 1
```

### Status Value Mapping
The `zte_onu_status` metric uses the following numeric values:

| Value | Status       |
|-------|--------------|
| `1`   | Online       |
| `2`   | Dying Gasp   |
| `3`   | LOS          |
| `4`   | PowerOff     |
| `0`   | Other/Unknown|

## License
[MIT License](https://github.com/megadata-dev/go-snmp-olt-zte-c320/blob/main/LICENSE)
