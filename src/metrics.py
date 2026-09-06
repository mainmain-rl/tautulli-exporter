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
    "Total bandwidth used by all current streams, in Kbps.",
)
BANDWIDTH_LAN_KBPS = Gauge(
    "tautulli_bandwidth_lan_kbps",
    "Bandwidth used by streams on the LAN, in Kbps.",
)
BANDWIDTH_WAN_KBPS = Gauge(
    "tautulli_bandwidth_wan_kbps",
    "Bandwidth used by streams over the WAN, in Kbps.",
)
SCRAPE_DURATION_SECONDS = Histogram(
    "tautulli_scrape_duration_seconds",
    "Time spent querying the Tautulli API for a single scrape.",
)
SCRAPE_ERRORS_TOTAL = Counter(
    "tautulli_scrape_errors_total",
    "Total number of failed scrapes of the Tautulli API.",
)


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


async def refresh(client: TautulliClient) -> None:
    """Fetch current activity from Tautulli and update all gauges in place.

    On failure, ``UP`` is set to 0, the error counter is incremented, and
    the other gauges are left untouched (they simply keep reporting their
    last known value rather than resetting to zero).
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
