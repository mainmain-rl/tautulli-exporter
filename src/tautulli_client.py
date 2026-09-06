"""Minimal async client for the Tautulli API.

Only wraps the single endpoint this exporter needs. See the official docs:
https://docs.tautulli.com/extending-tautulli/api-reference#get_activity
"""

from __future__ import annotations

from typing import Any

import httpx

from config import Settings


class TautulliAPIError(RuntimeError):
    """Raised when the Tautulli API responds but reports a non-success result."""


class TautulliClient:
    """Thin async wrapper around ``GET /api/v2?cmd=get_activity``."""

    def __init__(self, settings: Settings) -> None:
        self._apikey = settings.tautulli_apikey
        self._endpoint = f"{settings.tautulli_url}{settings.tautulli_base_path}/api/v2"
        self._http = httpx.AsyncClient(
            timeout=settings.tautulli_timeout,
            verify=settings.tautulli_verify_ssl,
        )

    async def aclose(self) -> None:
        """Release the underlying HTTP connection pool."""
        await self._http.aclose()

    async def get_activity(self) -> dict[str, Any]:
        """Fetch current PMS activity and return the ``data`` payload.

        Raises:
            httpx.HTTPError: on network/timeout/HTTP-status errors.
            TautulliAPIError: if Tautulli responds but ``result`` != "success".
        """
        params = {"apikey": self._apikey, "cmd": "get_activity"}
        response = await self._http.get(self._endpoint, params=params)
        response.raise_for_status()
        payload = response.json()

        envelope = payload.get("response", {})
        if envelope.get("result") != "success":
            raise TautulliAPIError(f"Tautulli API error: {envelope.get('message')}")

        return envelope.get("data") or {}
