# Tautulli Exporter (Go Version)

![alt text](images/tautulli_exporter_logo.png)

A Prometheus exporter for [Tautulli](https://tautulli.com/) written in Go. It calls Tautulli's `get_activity` API and exposes current Plex streaming activity (stream counts by type, bandwidth usage) as Prometheus metrics.

## Features

- **Lightweight**: Written in Go for minimal resource usage
- **Fast**: Efficient HTTP client and Prometheus metrics handling
- **Container-friendly**: Easy to deploy in Docker/Kubernetes
- **Health checks**: Built-in `/health` endpoint for monitoring
- **Environment-based configuration**: All settings via environment variables
- **Grafana Dashboard**: A functional [Grafana Dashboard](grafana-dashboard.json)

## Metrics Exposed

| Metric | Type | Description |
|---|---|---|
| `tautulli_up` | gauge | 1 if the last scrape of the Tautulli API succeeded, 0 otherwise |
| `tautulli_stream_count` | gauge | Total number of active streams |
| `tautulli_stream_count_transcode` | gauge | Streams currently being transcoded |
| `tautulli_stream_count_direct_play` | gauge | Streams in direct play |
| `tautulli_stream_count_direct_stream` | gauge | Streams in direct stream (remux) |
| `tautulli_bandwidth_total_kbps` | gauge | Total bandwidth across all streams, in Kbps |
| `tautulli_bandwidth_lan_kbps` | gauge | Bandwidth used by LAN streams, in Kbps |
| `tautulli_bandwidth_wan_kbps` | gauge | Bandwidth used by WAN streams, in Kbps |
| `tautulli_scrape_duration_seconds` | histogram | Time spent calling the Tautulli API per scrape |
| `tautulli_scrape_errors_total` | counter | Total number of failed scrapes |
| `tautulli_session_bandwidth_kbps` | gauge | Bandwidth by session (one time series per active session) |

## Configuration

All configuration is via environment variables. See [`.env.example`](.env.example) for the full list:

| Variable | Default | Description |
|---|---|---|
| `TAUTULLI_URL` | `http://127.0.0.1:8181` | Base URL of your Tautulli instance |
| `TAUTULLI_APIKEY` | — | **Required.** Tautulli API key (Settings → Web Interface) |
| `TAUTULLI_VERIFY_SSL` | `true` | Set to `false` for self-signed HTTPS |
| `LISTEN_PORT` | `9105` | Port the exporter listens on |
| `LISTEN_HOST` | `0.0.0.0` | Host the exporter binds to |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |

## Running with Docker

```bash
docker run -p 9105:9105 \
  -e TAUTULLI_URL=http://tautulli:8181 \
  -e TAUTULLI_APIKEY=your-api-key \
  ghcr.io/romain/tautulli-exporter:v1
```

## Kubernetes Deployment (Sidecar)

```yaml
- name: tautulli-exporter
  image: ghcr.io/romain/tautulli-exporter:latest
  ports:
    - name: metrics
      containerPort: 9105
  env:
    - name: TAUTULLI_URL
      value: "http://127.0.0.1:8181"
    - name: TAUTULLI_APIKEY
      valueFrom:
        secretKeyRef:
          name: tautulli-exporter
          key: apikey
  readinessProbe:
    httpGet: { path: /health, port: 9105 }
  livenessProbe:
    httpGet: { path: /health, port: 9105 }
```

## Building and Running Locally

```bash
# Build the Go application
go build -o tautulli-exporter .

# Run with environment variables
export TAUTULLI_APIKEY="your_api_key"
./tautulli-exporter

# Access metrics at http://localhost:9105/metrics
```

## Development

The Go implementation maintains the same functionality as the original Python version:
- Async HTTP client (using Go's concurrent model)
- Prometheus metrics with the same names and help text
- Health checks at `/health`
- Root endpoint at `/`
- Environment-based configuration
- Graceful shutdown handling

The main differences are:
- **Performance**: Go's native concurrency and lower memory footprint
- **Simplicity**: No external dependencies beyond Prometheus client library
- **Deployment**: Smaller container images and faster startup times
