package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Config holds the application configuration
type Config struct {
	TautulliURL       string
	TautulliAPIKey    string
	TautulliBasePath  string
	TautulliTimeout   time.Duration
	TautulliVerifySSL bool

	// Historical metrics settings
	HistoryEnable            bool
	HistoryGrouping          bool
	HistoryIncludeActivity   bool
	HistoryUser              string
	HistoryUserID            int
	HistoryStartDate         string
	HistoryBefore            string
	HistoryAfter             string
	HistorySectionID         int
	HistoryMediaType         string
	HistoryTranscodeDecision string
	HistoryOrderColumn       string
	HistoryOrderDirection    string
	HistoryStart             int
	HistoryLength            int
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
	Duration          interface{} `json:"duration"`          // in seconds
	BytesTransferred  interface{} `json:"bytes_transferred"` // total bytes transferred
}

// TautulliHistory represents the response from Tautulli's get_history API
type TautulliHistory struct {
	Response struct {
		Result  string          `json:"result"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	} `json:"response"`
}

// TautulliHistorySession represents a single session in history
type TautulliHistorySession struct {
	RatingKey         int         `json:"rating_key"`
	Title             string      `json:"title"`
	FullTitle         string      `json:"full_title"`
	GrandparentTitle  string      `json:"grandparent_title"`
	MediaType         string      `json:"media_type"`
	User              string      `json:"user"`
	Player            string      `json:"player"`
	Product           string      `json:"product"`
	Platform          string      `json:"platform"`
	TranscodeDecision interface{} `json:"transcode_decision"`
	State             string      `json:"state"`
	Location          string      `json:"location"`
	Duration          int         `json:"duration"`  // in seconds
	Bandwidth         int         `json:"bandwidth"` // in bytes
	AddedAt           int         `json:"added_at"`  // Unix timestamp
	Date              string      `json:"date"`      // formatted date
	QualityProfile    int         `json:"quality_profile"`
}

// HistoryStats represents aggregated statistics from get_history
type HistoryStats struct {
	TotalPlays      int             `json:"total_plays"`
	TotalDuration   float64         `json:"total_duration"`
	MostPlayedMedia string          `json:"most_played_media"`
	Users           []UserStats     `json:"users"`
	Platforms       []PlatformStats `json:"platforms"`
}

// UserStats represents user statistics from history
type UserStats struct {
	UserID   int     `json:"user_id"`
	Username string  `json:"username"`
	Plays    int     `json:"plays"`
	Duration float64 `json:"duration"`
}

// PlatformStats represents platform statistics from history
type PlatformStats struct {
	Platform string  `json:"platform"`
	Plays    int     `json:"plays"`
	Duration float64 `json:"duration"`
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
	defer func() { _ = resp.Body.Close() }()

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

// GetHistory fetches playback history from Tautulli
func (c *TautulliClient) GetHistory(ctx context.Context) (*TautulliHistory, error) {
	historyURL := fmt.Sprintf("%s%s/api/v2?apikey=%s&cmd=get_history",
		c.config.TautulliURL, c.config.TautulliBasePath, c.config.TautulliAPIKey)

	// Build query parameters
	params := url.Values{}

	if c.config.HistoryGrouping {
		params.Set("grouping", "1")
	}
	if c.config.HistoryIncludeActivity {
		params.Set("include_activity", "1")
	}
	if c.config.HistoryUser != "" {
		params.Set("user", c.config.HistoryUser)
	}
	if c.config.HistoryUserID > 0 {
		params.Set("user_id", fmt.Sprintf("%d", c.config.HistoryUserID))
	}
	if c.config.HistoryStartDate != "" {
		params.Set("start_date", c.config.HistoryStartDate)
	}
	if c.config.HistoryBefore != "" {
		params.Set("before", c.config.HistoryBefore)
	}
	if c.config.HistoryAfter != "" {
		params.Set("after", c.config.HistoryAfter)
	}
	if c.config.HistorySectionID > 0 {
		params.Set("section_id", fmt.Sprintf("%d", c.config.HistorySectionID))
	}
	if c.config.HistoryMediaType != "" {
		params.Set("media_type", c.config.HistoryMediaType)
	}
	if c.config.HistoryTranscodeDecision != "" {
		params.Set("transcode_decision", c.config.HistoryTranscodeDecision)
	}
	if c.config.HistoryOrderColumn != "" {
		params.Set("order_column", c.config.HistoryOrderColumn)
	}
	if c.config.HistoryOrderDirection != "" {
		params.Set("order_dir", c.config.HistoryOrderDirection)
	}
	if c.config.HistoryStart >= 0 {
		params.Set("start", fmt.Sprintf("%d", c.config.HistoryStart))
	}
	if c.config.HistoryLength > 0 {
		params.Set("length", fmt.Sprintf("%d", c.config.HistoryLength))
	}

	fullURL := historyURL + "&" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

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

	var history TautulliHistory
	if err := json.Unmarshal(apiResponse.Response.Data, &history); err != nil {
		return nil, fmt.Errorf("failed to decode history data: %w", err)
	}

	return &history, nil
}

// IsHistoryEnabled returns whether historical metrics are enabled
func (c *TautulliClient) IsHistoryEnabled() bool {
	return c.config.HistoryEnable
}

// GetHistoryStats fetches aggregated history statistics from Tautulli
func (c *TautulliClient) GetHistoryStats(ctx context.Context) (*HistoryStats, error) {
	history, err := c.GetHistory(ctx)
	if err != nil {
		return nil, err
	}

	stats := &HistoryStats{}

	// Parse the data array from history
	if history.Response.Data != nil {
		var sessions []TautulliHistorySession
		if err := json.Unmarshal(history.Response.Data, &sessions); err == nil {
			stats.TotalPlays = len(sessions)

			// Extract unique users and platforms
			userMap := make(map[string]bool)
			platformMap := make(map[string]bool)

			for _, session := range sessions {
				if session.User != "" {
					userMap[session.User] = true
				}
				if session.Platform != "" {
					platformMap[session.Platform] = true
				}
			}

			stats.Users = make([]UserStats, 0, len(userMap))
			for user := range userMap {
				stats.Users = append(stats.Users, UserStats{
					Username: user,
				})
			}

			stats.Platforms = make([]PlatformStats, 0, len(platformMap))
			for platform := range platformMap {
				stats.Platforms = append(stats.Platforms, PlatformStats{
					Platform: platform,
				})
			}

			// Calculate total duration
			var totalDuration float64
			for _, session := range sessions {
				if session.Duration > 0 {
					totalDuration += float64(session.Duration)
				}
			}
			stats.TotalDuration = totalDuration
		}
	}

	return stats, nil
}
