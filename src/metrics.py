"""Prometheus metric definitions and the logic that refreshes them.

Metrics are module-level singletons registered on prometheus_client's
default global registry, as is conventional for a single-process exporter.
"""

from __future__ import annotations

import logging
import time
from typing import Any

from prometheus_client import Counter, Gauge, Histogram

from tautulli_client import TautulliClient

logger = logging.getLogger(__name__)


UP = Gauge(
    "tautulli_up",
    "Whether the last scrape of the Tautulli API succeeded (1) or failed (0).",
)
STREAM_COUNT = Gauge(
    "tautulli_stream_count",
    "Total number of active streams.",
)
STREAM_COUNT_TRANSCODE = Gauge(
    "tautulli_stream_count_transcode",
    "Number of active streams currently being transcoded.",
)
STREAM_COUNT_DIRECT_PLAY = Gauge(
    "tautulli_stream_count_direct_play",
    "Number of active direct play streams.",
)
STREAM_COUNT_DIRECT_STREAM = Gauge(
    "tautulli_stream_count_direct_stream",
    "Number of active direct stream (remuxed) streams.",
)
BANDWIDTH_TOTAL_KBPS = Gauge(
    "tautulli_bandwidth_total_kbps",
    "Total live bandwidth used by all current streams, in Kbps (instantaneous, not cumulative).",
)
BANDWIDTH_LAN_KBPS = Gauge(
    "tautulli_bandwidth_lan_kbps",
    "Live bandwidth used by streams on the LAN, in Kbps (instantaneous, not cumulative).",
)
BANDWIDTH_WAN_KBPS = Gauge(
    "tautulli_bandwidth_wan_kbps",
    "Live bandwidth used by streams over the WAN, in Kbps (instantaneous, not cumulative).",
)
SCRAPE_DURATION_SECONDS = Histogram(
    "tautulli_scrape_duration_seconds",
    "Time spent querying the Tautulli API for a single scrape.",
)
SCRAPE_ERRORS_TOTAL = Counter(
    "tautulli_scrape_errors_total",
    "Total number of failed scrapes of the Tautulli API.",
)
SESSION_BANDWIDTH_KBPS = Gauge(
    "tautulli_session_bandwidth_kbps",
    "Live bandwidth of a single active stream, in Kbps. "
    "One time series per currently active session; use `sum by (user)`, "
    "`sum by (transcode_decision)`, etc. to break it down.",
    labelnames=[
        "session_key",
        "user",
        "player",
        "product",
        "platform",
        "transcode_decision",
        "state",
        "location",
        "media_type",
        "title",
    ],
)
_TRANSCODE_DECISION_LABELS = {
    "direct play": "direct_play",
    "copy": "direct_stream",
    "transcode": "transcode",
}


def _as_int(value: Any, default: int = 0) -> int:
    """Normalise a value coming from Tautulli into an int.

    The `get_activity` payload mixes real ints and numeric strings for the
    same kind of field (e.g. ``stream_count`` is a string while
    ``stream_count_direct_play`` is an int), so every value is coerced
    defensively here.
    """
    try:
        return int(value)
    except (TypeError, ValueError):
        try:
            return int(float(value))
        except (TypeError, ValueError):
            return default


def _normalize_transcode_decision(value: Any) -> str:
    """Map Tautulli's raw transcode_decision to a stable, readable label value."""
    text = str(value or "").strip().lower()
    return _TRANSCODE_DECISION_LABELS.get(text, text or "unknown")


def _session_title(session: dict[str, Any]) -> str:
    """Build a human-readable title, handling TV episodes vs. everything else."""
    if session.get("media_type") == "episode":
        show = session.get("grandparent_title") or ""
        episode = session.get("title") or ""
        combined = f"{show} - {episode}".strip(" -")
        if combined:
            return combined
    return session.get("full_title") or session.get("title") or "unknown"


def _session_labels(session: dict[str, Any]) -> dict[str, str]:
    """Extract the label set for one session, defaulting anything missing."""
    return {
        "session_key": str(session.get("session_key") or session.get("session_id") or "unknown"),
        "user": str(session.get("user") or session.get("friendly_name") or "unknown"),
        "player": str(session.get("player") or "unknown"),
        "product": str(session.get("product") or "unknown"),
        "platform": str(session.get("platform") or "unknown"),
        "transcode_decision": _normalize_transcode_decision(session.get("transcode_decision")),
        "state": str(session.get("state") or "unknown"),
        "location": str(session.get("location") or "unknown"),
        "media_type": str(session.get("media_type") or "unknown"),
        "title": _session_title(session),
    }


def _refresh_session_metrics(sessions: list[dict[str, Any]]) -> None:
    """Rebuild the per-session gauge from scratch on every scrape.

    prometheus_client has no way to remove a single labelled series, only
    to clear *all* of them (`Gauge.clear()`). So instead of trying to track
    which sessions ended since the last scrape, we simply wipe the whole
    metric and re-populate it from the current session list every time.
    This guarantees a session that has ended never lingers in `/metrics`
    with a stale "still streaming" value.
    """
    SESSION_BANDWIDTH_KBPS.clear()
    for session in sessions:
        labels = _session_labels(session)
        SESSION_BANDWIDTH_KBPS.labels(**labels).set(_as_int(session.get("bandwidth")))


async def refresh(client: TautulliClient) -> None:
    """Fetch current activity from Tautulli and update all metrics in place.

    On failure, ``UP`` is set to 0 and the error counter is incremented.
    The other metrics (aggregate gauges *and* per-session series) are left
    untouched, simply reporting their last known value -- this is the
    standard Prometheus exporter pattern (e.g. blackbox_exporter): always
    gate dashboards/alerts on `tautulli_up` alongside these.
    """
    start = time.perf_counter()
    try:
        data = await client.get_activity()
    except Exception:
        logger.exception("Failed to fetch activity from Tautulli")
        UP.set(0)
        SCRAPE_ERRORS_TOTAL.inc()
        return
    finally:
        SCRAPE_DURATION_SECONDS.observe(time.perf_counter() - start)

    UP.set(1)
    STREAM_COUNT.set(_as_int(data.get("stream_count")))
    STREAM_COUNT_TRANSCODE.set(_as_int(data.get("stream_count_transcode")))
    STREAM_COUNT_DIRECT_PLAY.set(_as_int(data.get("stream_count_direct_play")))
    STREAM_COUNT_DIRECT_STREAM.set(_as_int(data.get("stream_count_direct_stream")))
    BANDWIDTH_TOTAL_KBPS.set(_as_int(data.get("total_bandwidth")))
    BANDWIDTH_LAN_KBPS.set(_as_int(data.get("lan_bandwidth")))
    BANDWIDTH_WAN_KBPS.set(_as_int(data.get("wan_bandwidth")))

    _refresh_session_metrics(data.get("sessions") or [])
