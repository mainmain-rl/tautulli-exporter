package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Config holds the application configuration
type Config struct {
	TautulliURL       string
	TautulliAPIKey    string
	TautulliBasePath  string
	TautulliTimeout   time.Duration
	TautulliVerifySSL bool
}

// TautulliActivity represents the response from Tautulli's get_activity API
type TautulliActivity struct {
	StreamCount             interface{} `json:"stream_count"`
	StreamCountTranscode    interface{} `json:"stream_count_transcode"`
	StreamCountDirectPlay   interface{} `json:"stream_count_direct_play"`
	StreamCountDirectStream interface{} `json:"stream_count_direct_stream"`
	TotalBandwidth          interface{} `json:"total_bandwidth"`
	LanBandwidth            interface{} `json:"lan_bandwidth"`
	WanBandwidth            interface{} `json:"wan_bandwidth"`
	Sessions                []Session   `json:"sessions"`
}

// Session represents a single streaming session
type Session struct {
	SessionKey        interface{} `json:"session_key"`
	SessionID         interface{} `json:"session_id"`
	User              string      `json:"user"`
	FriendlyName      string      `json:"friendly_name"`
	Player            string      `json:"player"`
	Product           string      `json:"product"`
	Platform          string      `json:"platform"`
	TranscodeDecision interface{} `json:"transcode_decision"`
	State             string      `json:"state"`
	Location          interface{} `json:"location"`
	MediaType         string      `json:"media_type"`
	GrandparentTitle  string      `json:"grandparent_title"`
	Title             string      `json:"title"`
	FullTitle         string      `json:"full_title"`
	Bandwidth         interface{} `json:"bandwidth"`
	IpAddress         string      `json:"ip_address"`
}

// TautulliClient handles communication with the Tautulli API
type TautulliClient struct {
	config Config
	client *http.Client
}

// NewTautulliClient creates a new Tautulli client
func NewTautulliClient(config Config) *TautulliClient {
	transport := &http.Transport{
		DisableKeepAlives: true,
	}
	if !config.TautulliVerifySSL {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &TautulliClient{
		config: config,
		client: &http.Client{
			Transport: transport,
			Timeout:   config.TautulliTimeout,
		},
	}
}

// APIResponse wraps Tautulli's generic API envelope
type APIResponse struct {
	Response struct {
		Result  string          `json:"result"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	} `json:"response"`
}

// GetActivity fetches current activity from Tautulli
func (c *TautulliClient) GetActivity(ctx context.Context) (*TautulliActivity, error) {
	url := fmt.Sprintf("%s%s/api/v2?apikey=%s&cmd=get_activity",
		c.config.TautulliURL, c.config.TautulliBasePath, c.config.TautulliAPIKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResponse APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResponse.Response.Result != "success" {
		return nil, fmt.Errorf("tautulli API error: %s", apiResponse.Response.Message)
	}

	var activity TautulliActivity
	if err := json.Unmarshal(apiResponse.Response.Data, &activity); err != nil {
		return nil, fmt.Errorf("failed to decode activity data: %w", err)
	}

	return &activity, nil
}

// asInt converts an interface{} value to int for testing purposes
func asInt(value interface{}, defaultValue int) int {
	if value == nil {
		return defaultValue
	}

	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
		return defaultValue
	default:
		return defaultValue
	}
}
