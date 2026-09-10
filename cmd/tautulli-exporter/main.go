package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gin "github.com/gin-gonic/gin"

	"tautulli-exporter/internal/client"
	"tautulli-exporter/internal/config"
	"tautulli-exporter/internal/metrics"
)

var Version string = "2.0.0"

// HealthCheckHandler handles health checks
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": Version,
	})
}

// RootHandler handles root requests
func RootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":      "tautulli-exporter",
		"metrics_path": "/metrics",
		"health_path":  "/health",
	})
}

func main() {
	// Load configuration
	cfg := config.LoadConfig(Version)

	// Validate required configuration
	if cfg.TautulliAPIKey == "" {
		log.Fatal("TAUTULLI_APIKEY environment variable is required")
	}

	// Create Tautulli client
	tautulliConfig := client.Config{
		TautulliURL:       cfg.TautulliURL,
		TautulliAPIKey:    cfg.TautulliAPIKey,
		TautulliBasePath:  cfg.TautulliBasePath,
		TautulliTimeout:   cfg.TautulliTimeout,
		TautulliVerifySSL: cfg.TautulliVerifySSL,
	}
	tautulliClient := client.NewTautulliClient(tautulliConfig)

	// Set up Gin HTTP server
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Metrics endpoint
	router.GET("/metrics", func(c *gin.Context) {
		// Call the metrics handler
		metrics.MetricsHandler(tautulliClient)(c.Writer, c.Request)
	})

	// Health endpoint
	router.GET("/health", HealthCheckHandler)

	// Root endpoint
	router.GET("/", RootHandler)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Print("Thanks to use Tautulli Exporter ! (https://github.com/mainmain-rl/tautulli-exporter)")
	log.Printf("Starting tautulli-exporter: %s", cfg.Version)
	log.Printf("Tautulli instance target: %s", cfg.TautulliURL)
	log.Printf("HTTP server listening on port: %d", cfg.ListenPort)

	// Start server in goroutine
	server := &http.Server{Addr: fmt.Sprintf("%s:%d", cfg.ListenHost, cfg.ListenPort), Handler: router}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-sigChan
	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("tautulli-exporter shut down cleanly")
}
