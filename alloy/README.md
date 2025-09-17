# Grafana Alloy Configuration

This directory contains the configuration for Grafana Alloy, a telemetry collector that complements your existing OpenTelemetry stack.

## Overview

Grafana Alloy is configured to collect:
- **Metrics**: System metrics from Node Exporter and application metrics
- **Logs**: System logs, Docker container logs, and application logs
- **Traces**: OTLP traces from your applications

All collected telemetry is forwarded to your existing stack:
- Metrics → Prometheus
- Logs → Loki
- Traces → Tempo

## Configuration Files

- `alloy-config.alloy`: Main Alloy configuration file
- `README.md`: This documentation file

## Key Features

### Metrics Collection
- Scrapes Node Exporter metrics
- Collects Alloy's own internal metrics
- Supports application metrics (if exposed on `/metrics` endpoint)
- Remote writes to Prometheus

### Logs Collection
- System logs from `/var/log/syslog`, `/var/log/auth.log`, `/var/log/kern.log`
- Docker container logs via Docker socket
- Application logs from `/tmp/go-app.log`
- Forwards to Loki

### Traces Collection
- OTLP receiver on ports 4317 (gRPC) and 4318 (HTTP)
- Processes traces with batching
- Forwards to Tempo

## Accessing Alloy

- **Web UI**: http://localhost:12345
- **Metrics Endpoint**: http://localhost:12345/metrics
- **Health Check**: http://localhost:12345/-/ready

## Integration with Existing Stack

Alloy works alongside your existing OpenTelemetry Collector:
- **OpenTelemetry Collector**: Handles application traces, metrics, and logs
- **Alloy**: Provides additional system monitoring and log collection

## Monitoring Alloy

Alloy's own metrics are scraped by Prometheus and can be visualized in Grafana using the `alloy` job.

## Configuration Updates

To modify the Alloy configuration:
1. Edit `alloy-config.alloy`
2. Restart the alloy service: `docker-compose restart alloy`

## Troubleshooting

- Check Alloy logs: `docker-compose logs alloy`
- Verify health: `curl http://localhost:12345/-/ready`
- View metrics: `curl http://localhost:12345/metrics`

## Ports Used

- `12345`: Alloy web UI and metrics endpoint (exposed externally)
