# tautulli-exporter

A small Prometheus exporter for [Tautulli](https://tautulli.com/). It calls
Tautulli's `get_activity` API and exposes current Plex streaming activity
(stream counts by type, bandwidth usage) as Prometheus metrics.

Built with FastAPI + `prometheus_client`, packaged as a container image
meant to run as a sidecar next to your Tautulli container.

## Metrics

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

If a scrape fails, `tautulli_up` drops to 0 but the other gauges keep their
last known value (the standard Prometheus exporter pattern) — always gate
alerts/dashboards on `tautulli_up` alongside them.

## Configuration

All configuration is via environment variables. See [`.env.example`](.env.example)
for the full list; the important ones:

| Variable | Default | Description |
|---|---|---|
| `TAUTULLI_URL` | `http://127.0.0.1:8181` | Base URL of your Tautulli instance |
| `TAUTULLI_APIKEY` | — | **Required.** Tautulli API key (Settings → Web Interface) |
| `TAUTULLI_VERIFY_SSL` | `true` | Set to `false` for self-signed HTTPS |
| `LISTEN_PORT` | `9105` | Port the exporter listens on |

The process fails fast at startup if `TAUTULLI_APIKEY` is missing.

## Running it

**With Docker:**

```bash
docker run -p 9105:9105 \
  -e TAUTULLI_URL=http://tautulli:8181 \
  -e TAUTULLI_APIKEY=your-api-key \
  ghcr.io/<your-org>/tautulli-exporter:latest
```

**As a Kubernetes sidecar** (same pod as Tautulli, calling it over
`127.0.0.1`):

```yaml
- name: tautulli-exporter
  image: ghcr.io/<your-org>/tautulli-exporter:latest
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

Then scrape `<service>:9105/metrics` from Prometheus as usual (e.g. via a
`ScrapeConfig` / `PodMonitor` if you're using the Prometheus Operator).
