# Build stage
FROM golang:1.25.6-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download
RUN go mod verify

# Copy source code
COPY internal/. internal/.
COPY cmd/. cmd/.

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o tautulli-exporter ./cmd/tautulli-exporter

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/tautulli-exporter .

# Expose the metrics port
EXPOSE 9105

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:9105/health

# Default environment variables
ENV LISTEN_PORT=9105 \
    LOG_LEVEL=info

# Command to run the application
CMD ["./tautulli-exporter"]
