FROM python:3.13-slim

WORKDIR /app

# Installation de uv pour une gestion rapide des dépendances
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# Copier les fichiers de dépendances
COPY pyproject.toml .
COPY .python-version .

# Installer les dépendances
RUN uv sync --no-dev --frozen || uv sync --no-dev

# Copier le code source
COPY src/ src/

# Port exposé
EXPOSE 9105

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["python", "-c", "import os, urllib.request; urllib.request.urlopen('http://127.0.0.1:' + os.environ.get('LISTEN_PORT', '9105') + '/health', timeout=3)"]

ENV PYTHONPATH=/app/src
ENV LISTEN_PORT=9105
ENV LOG_LEVEL=info

CMD ["sh", "-c", "exec uv run python -m uvicorn src.main:app --host 0.0.0.0 --port ${LISTEN_PORT} --log-level ${LOG_LEVEL}"]
