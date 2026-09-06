"""Runtime configuration for the exporter.

All settings are read from environment variables (case-insensitive) or from
a local ``.env`` file during development. See ``.env.example`` for the full
list of supported variables.
"""

from __future__ import annotations

from pydantic import field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Exporter configuration.

    ``tautulli_apikey`` has no default on purpose: the process should fail
    fast at startup with a clear error rather than silently exposing an
    exporter that will fail on every scrape.
    """

    # --- Tautulli connection ---
    tautulli_url: str = "http://127.0.0.1:8181"
    # Set this if Tautulli is served behind a custom HTTP_ROOT, e.g. "/tautulli"
    tautulli_base_path: str = ""
    tautulli_apikey: str
    tautulli_timeout: float = 5.0
    # Disable only if Tautulli uses a self-signed certificate you trust
    tautulli_verify_ssl: bool = True

    # --- HTTP server ---
    listen_host: str = "0.0.0.0"
    listen_port: int = 9105
    log_level: str = "INFO"

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    @field_validator("tautulli_url")
    @classmethod
    def _strip_trailing_slash(cls, value: str) -> str:
        return value.rstrip("/")

    @field_validator("tautulli_base_path")
    @classmethod
    def _strip_trailing_slash_base_path(cls, value: str) -> str:
        return value.rstrip("/")


def get_settings() -> Settings:
    """Build a fresh Settings instance.

    Kept as a small function (instead of a bare module-level constant) so it
    can be swapped out easily in tests via monkeypatching or dependency
    overrides if needed.
    """
    return Settings()
