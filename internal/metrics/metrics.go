package metrics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"tautulli-exporter/internal/client"
)

// Metric names as constants
const (
	MetricUp                      = "tautulli_up"
	MetricStreamCount             = "tautulli_stream_count"
	MetricStreamCountTranscode    = "tautulli_stream_count_transcode"
	MetricStreamCountDirectPlay   = "tautulli_stream_count_direct_play"
	MetricStreamCountDirectStream = "tautulli_stream_count_direct_stream"
	MetricBandwidthTotalKbps      = "tautulli_bandwidth_total_kbps"
	MetricBandwidthLanKbps        = "tautulli_bandwidth_lan_kbps"
	MetricBandwidthWanKbps        = "tautulli_bandwidth_wan_kbps"
	MetricScrapeDuration          = "tautulli_scrape_duration_seconds"
	MetricScrapeErrorsTotal       = "tautulli_scrape_errors_total"
	MetricSessionBandwidthKbps    = "tautulli_session_bandwidth_kbps"
)

// Metrics holds all Prometheus metrics
var (
	up = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: MetricUp,
		Help: "Whether the last scrape of the Tautulli API succeeded (1) or failed (0).",
	})

	streamCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: MetricStreamCount,
		Help: "Total number of active streams.",
	})

	streamCountTranscode = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: MetricStreamCountTranscode,
		Help: "Number of active streams currently being transcoded.",
	})

	streamCountDirectPlay = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: MetricStreamCountDirectPlay,
		Help: "Number of active direct play streams.",
	})

	streamCountDirectStream = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: MetricStreamCountDirectStream,
		Help: "Number of active direct stream (remuxed) streams.",
	})

	bandwidthTotalKbps = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: MetricBandwidthTotalKbps,
		Help: "Total live bandwidth used by all current streams, in Kbps (instantaneous, not cumulative).",
	})

	bandwidthLanKbps = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: MetricBandwidthLanKbps,
		Help: "Live bandwidth used by streams on the LAN, in Kbps (instantaneous, not cumulative).",
	})

	bandwidthWanKbps = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: MetricBandwidthWanKbps,
		Help: "Live bandwidth used by streams over the WAN, in Kbps (instantaneous, not cumulative).",
	})

	scrapeDurationSeconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: MetricScrapeDuration,
		Help: "Time spent querying the Tautulli API for a single scrape.",
	})

	scrapeErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: MetricScrapeErrorsTotal,
		Help: "Total number of failed scrapes of the Tautulli API.",
	})

	sessionBandwidthKbps = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: MetricSessionBandwidthKbps,
		Help: "Live bandwidth of a single active stream, in Kbps. One time series per currently active session; use `sum by (user)`, `sum by (transcode_decision)`, etc. to break it down.",
	}, []string{"session_key", "user", "player", "product", "platform", "transcode_decision", "state", "location", "media_type", "title", "ip_address"})
)

// init registers metrics with Prometheus
func init() {
	prometheus.MustRegister(up)
	prometheus.MustRegister(streamCount)
	prometheus.MustRegister(streamCountTranscode)
	prometheus.MustRegister(streamCountDirectPlay)
	prometheus.MustRegister(streamCountDirectStream)
	prometheus.MustRegister(bandwidthTotalKbps)
	prometheus.MustRegister(bandwidthLanKbps)
	prometheus.MustRegister(bandwidthWanKbps)
	prometheus.MustRegister(scrapeDurationSeconds)
	prometheus.MustRegister(scrapeErrorsTotal)
	prometheus.MustRegister(sessionBandwidthKbps)
}

// asFloat64 converts various types to float64 with a default
func asFloat64(value interface{}, defaultValue float64) float64 {
	if value == nil {
		return defaultValue
	}

	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
		return defaultValue
	default:
		return defaultValue
	}
}

// asInt converts various types to int with a default
func asInt(value interface{}, defaultValue int) int {
	if value == nil {
		return defaultValue
	}

	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
		return defaultValue
	default:
		return defaultValue
	}
}

// normalizeTranscodeDecision maps Tautulli's raw values to stable labels
func normalizeTranscodeDecision(value interface{}) string {
	if value == nil {
		return "unknown"
	}

	switch v := value.(type) {
	case string:
		text := strings.ToLower(strings.TrimSpace(v))
		switch text {
		case "direct play":
			return "direct_play"
		case "copy":
			return "direct_stream"
		case "transcode":
			return "transcode"
		default:
			if text != "" {
				return text
			}
		}
	default:
		return "unknown"
	}
	return "unknown"
}

// sessionTitle builds a human-readable title
func sessionTitle(session client.Session) string {
	if session.MediaType == "episode" && session.GrandparentTitle != "" && session.Title != "" {
		title := fmt.Sprintf("%s - %s", session.GrandparentTitle, session.Title)
		title = strings.Trim(title, " -")
		if title != "" {
			return title
		}
	}
	if session.FullTitle != "" {
		return session.FullTitle
	}
	if session.Title != "" {
		return session.Title
	}
	return "unknown"
}

// sessionLabels extracts labels for a session
func sessionLabels(session client.Session) prometheus.Labels {
	return prometheus.Labels{
		"session_key":        fmt.Sprintf("%v", session.SessionKey),
		"user":               session.User,
		"player":             session.Player,
		"product":            session.Product,
		"platform":           session.Platform,
		"transcode_decision": normalizeTranscodeDecision(session.TranscodeDecision),
		"state":              session.State,
		"location":           fmt.Sprintf("%v", session.Location),
		"media_type":         session.MediaType,
		"title":              sessionTitle(session),
		"ip_address":         session.IpAddress,
	}
}

// RefreshMetrics updates all metrics from Tautulli data
func RefreshMetrics(client *client.TautulliClient) {
	start := time.Now()
	defer func() {
		scrapeDurationSeconds.Observe(time.Since(start).Seconds())
	}()

	activity, err := client.GetActivity(context.Background())
	if err != nil {
		log.Printf("Failed to fetch activity from Tautulli: %v", err)
		up.Set(0)
		scrapeErrorsTotal.Inc()
		return
	}

	up.Set(1)
	streamCount.Set(float64(asInt(activity.StreamCount, 0)))
	streamCountTranscode.Set(float64(asInt(activity.StreamCountTranscode, 0)))
	streamCountDirectPlay.Set(float64(asInt(activity.StreamCountDirectPlay, 0)))
	streamCountDirectStream.Set(float64(asInt(activity.StreamCountDirectStream, 0)))
	bandwidthTotalKbps.Set(asFloat64(activity.TotalBandwidth, 0))
	bandwidthLanKbps.Set(asFloat64(activity.LanBandwidth, 0))
	bandwidthWanKbps.Set(asFloat64(activity.WanBandwidth, 0))

	// Refresh session metrics
	sessionBandwidthKbps.Reset()
	for _, session := range activity.Sessions {
		labels := sessionLabels(session)
		bandwidth := asInt(session.Bandwidth, 0)
		sessionBandwidthKbps.With(labels).Set(float64(bandwidth))
	}
}

// MetricsHandler handles metrics requests
func MetricsHandler(client *client.TautulliClient) http.HandlerFunc {
	handler := promhttp.Handler()
	return func(w http.ResponseWriter, r *http.Request) {
		RefreshMetrics(client)
		handler.ServeHTTP(w, r)
	}
}
