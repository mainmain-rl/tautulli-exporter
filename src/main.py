"""FastAPI application exposing Tautulli activity as Prometheus metrics.

Design choice: metrics are refreshed on every call to ``/metrics`` (a "pull
through" model) rather than on a background timer. Tautulli's
``get_activity`` call is cheap and local (the exporter is meant to run as a
sidecar next to Tautulli), so this keeps the code simple and guarantees the
data is always as fresh as the last Prometheus scrape.
"""

from __future__ import annotations
from .version import __version__
import logging
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from fastapi import FastAPI, Response
from prometheus_client import CONTENT_TYPE_LATEST, generate_latest

from metrics import refresh
from config import get_settings
from tautulli_client import TautulliClient

settings = get_settings()

logging.basicConfig(
    level=settings.log_level.upper(),
    format="%(asctime)s %(levelname)s %(name)s: %(message)s",
)
logger = logging.getLogger(__name__)
logging.getLogger("httpx").setLevel(logging.WARNING)
logging.getLogger("httpcore").setLevel(logging.WARNING)

class HealthCheckFilter(logging.Filter):
    def filter(self, record: logging.LogRecord) -> bool:
        # Returns False to drop logs containing "/health"
        return record.getMessage().find("/health") == -1

logging.getLogger("uvicorn.access").addFilter(HealthCheckFilter())

@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    """Create the shared Tautulli HTTP client on startup, close it on shutdown."""
    app.state.tautulli_client = TautulliClient(settings)
    logger.info("tautulli-exporter (%s) starting up (target=%s)", __version__, settings.tautulli_url)
    try:
        yield
    finally:
        await app.state.tautulli_client.aclose()
        logger.info("tautulli-exporter shut down cleanly")


app = FastAPI(
    title="Tautulli Prometheus Exporter",
    description="Exposes live Tautulli/Plex streaming activity as Prometheus metrics.",
    version=__version__,
    lifespan=lifespan,
    openapi_url=None,
    docs_url=None,
    redoc_url=None,
)


@app.get("/metrics", include_in_schema=False)
async def get_metrics() -> Response:
    """Scrape endpoint: refresh metrics from Tautulli, then render Prometheus text format."""
    client: TautulliClient = app.state.tautulli_client
    await refresh(client)
    return Response(content=generate_latest(), media_type=CONTENT_TYPE_LATEST)


@app.get("/health", include_in_schema=False)
async def healthz() -> dict[str, str]:
    """Liveness probe for the exporter process itself (does not call Tautulli)."""
    return {"status": "ok", "version": app.version}


@app.get("/", include_in_schema=False)
async def root() -> dict[str, str]:
    """Tiny landing payload so hitting the root isn't a dead end."""
    return {"service": "tautulli-exporter", "metrics_path": "/metrics", "health_path": "/health"}
