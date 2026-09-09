package metrics

import (
	"tautulli-exporter/internal/client"
	"testing"
)

func TestAsFloat64(t *testing.T) {
	tests := []struct {
		name         string
		value        interface{}
		expected     float64
		defaultValue float64
	}{
		{"nil value", nil, 0.0, 0.0},
		{"float64 value", float64(123.45), 123.45, 0.0},
		{"int value", int(123), 123.0, 0.0},
		{"string number", "678.9", 678.9, 0.0},
		{"invalid string", "abc", 0.0, 0.0},
		{"string with default", "abc", -1.0, -1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := asFloat64(tt.value, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAsInt(t *testing.T) {
	tests := []struct {
		name         string
		value        interface{}
		expected     int
		defaultValue int
	}{
		{"nil value", nil, 0, 0},
		{"int value", int(123), 123, 0},
		{"float64 value", float64(123.7), 123, 0},
		{"string number", "456", 456, 0},
		{"invalid string", "abc", 0, 0},
		{"string with default", "abc", -1, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := asInt(tt.value, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNormalizeTranscodeDecision(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"nil value", nil, "unknown"},
		{"direct play string", "Direct Play", "direct_play"},
		{"copy string", "Copy", "direct_stream"},
		{"transcode string", "Transcode", "transcode"},
		{"other string", "Something Else", "something else"},
		{"empty string", "", "unknown"},
		{"whitespace string", "   ", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTranscodeDecision(tt.value)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSessionTitle(t *testing.T) {
	tests := []struct {
		name     string
		session  client.Session
		expected string
	}{
		{"episode with grandparent and title", client.Session{
			MediaType:        "episode",
			GrandparentTitle: "Series Name",
			Title:            "Episode Title",
			FullTitle:        "",
		}, "Series Name - Episode Title"},
		{"episode with full title", client.Session{
			MediaType:        "episode",
			GrandparentTitle: "",
			Title:            "",
			FullTitle:        "Full Episode Title",
		}, "Full Episode Title"},
		{"movie with title", client.Session{
			MediaType:        "movie",
			GrandparentTitle: "",
			Title:            "Movie Title",
			FullTitle:        "",
		}, "Movie Title"},
		{"empty title", client.Session{
			MediaType:        "movie",
			GrandparentTitle: "",
			Title:            "",
			FullTitle:        "",
		}, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sessionTitle(tt.session)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSessionLabels(t *testing.T) {
	session := client.Session{
		SessionKey:        123,
		User:              "testuser",
		Player:            "Chrome",
		Product:           "Plex Web",
		Platform:          "Web",
		TranscodeDecision: "direct play",
		State:             "playing",
		Location:          "local",
		MediaType:         "movie",
		Title:             "Test Movie",
		IpAddress:         "1.1.1.1",
	}

	labels := sessionLabels(session)

	if labels["session_key"] != "123" {
		t.Errorf("Expected session_key '123', got '%s'", labels["session_key"])
	}

	if labels["user"] != "testuser" {
		t.Errorf("Expected user 'testuser', got '%s'", labels["user"])
	}

	if labels["transcode_decision"] != "direct_play" {
		t.Errorf("Expected transcode_decision 'direct_play', got '%s'", labels["transcode_decision"])
	}

	if labels["title"] != "Test Movie" {
		t.Errorf("Expected title 'Test Movie', got '%s'", labels["title"])
	}
	if labels["ip_address"] != "1.1.1.1" {
		t.Errorf("Expected title '1.1.1.1', got '%s'", labels["title"])
	}
}
