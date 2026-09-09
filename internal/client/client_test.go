package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewTautulliClient(t *testing.T) {
	config := Config{
		TautulliURL:       "http://localhost:8181",
		TautulliAPIKey:    "test-key",
		TautulliBasePath:  "/base",
		TautulliTimeout:   5 * time.Second,
		TautulliVerifySSL: true,
	}

	client := NewTautulliClient(config)
	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.config.TautulliURL != config.TautulliURL {
		t.Errorf("Expected URL %s, got %s", config.TautulliURL, client.config.TautulliURL)
	}

	if client.client == nil {
		t.Fatal("Expected non-nil HTTP client")
	}
}

func TestGetActivitySuccess(t *testing.T) {
	// Create a test server that returns successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"response": map[string]interface{}{
				"result":  "success",
				"message": "Success",
				"data": map[string]interface{}{
					"stream_count":               2,
					"stream_count_transcode":     1,
					"stream_count_direct_play":   0,
					"stream_count_direct_stream": 1,
					"total_bandwidth":            500.5,
					"lan_bandwidth":              200.3,
					"wan_bandwidth":              300.2,
					"sessions": []interface{}{
						map[string]interface{}{
							"session_key":        123,
							"user":               "testuser",
							"player":             "Chrome",
							"product":            "Plex Web",
							"platform":           "Web",
							"transcode_decision": "direct play",
							"state":              "playing",
							"media_type":         "movie",
							"title":              "Test Movie",
							"bandwidth":          250.5,
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := Config{
		TautulliURL:       "http://" + server.Listener.Addr().String(),
		TautulliAPIKey:    "test-key",
		TautulliBasePath:  "/base",
		TautulliTimeout:   5 * time.Second,
		TautulliVerifySSL: true,
	}

	client := NewTautulliClient(config)
	activity, err := client.GetActivity(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if activity == nil {
		t.Fatal("Expected non-nil activity")
	}

	if asInt(activity.StreamCount, 0) != 2 {
		t.Errorf("Expected stream count 2, got %v", asInt(activity.StreamCount, 0))
	}

	if len(activity.Sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(activity.Sessions))
	}

	if activity.Sessions[0].User != "testuser" {
		t.Errorf("Expected user 'testuser', got '%s'", activity.Sessions[0].User)
	}
}

func TestGetActivityError(t *testing.T) {
	// Create a test server that returns error response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"response": map[string]interface{}{
				"result":  "error",
				"message": "API key invalid",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := Config{
		TautulliURL:       "http://" + server.Listener.Addr().String(),
		TautulliAPIKey:    "invalid-key",
		TautulliBasePath:  "/base",
		TautulliTimeout:   5 * time.Second,
		TautulliVerifySSL: true,
	}

	client := NewTautulliClient(config)
	activity, err := client.GetActivity(context.Background())

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if activity != nil {
		t.Fatal("Expected nil activity on error")
	}

	if err.Error() != "tautulli API error: API key invalid" {
		t.Errorf("Expected specific error message, got %v", err.Error())
	}
}

func TestGetActivityTimeout(t *testing.T) {
	// Create a test server that times out
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Longer than our timeout
	}))
	defer server.Close()

	config := Config{
		TautulliURL:       server.URL,
		TautulliAPIKey:    "test-key",
		TautulliBasePath:  "/base",
		TautulliTimeout:   1 * time.Second, // Short timeout
		TautulliVerifySSL: true,
	}

	client := NewTautulliClient(config)
	_, err := client.GetActivity(context.Background())

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	if err.Error() != "failed to make request: Get \"http://127.0.0.1:8080/base/api/v2?apikey=test-key&cmd=get_activity\": context deadline exceeded" {
		t.Logf("Got timeout error: %v", err)
	}
}
