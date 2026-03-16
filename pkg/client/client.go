// Package client provides HTTP client for Clink API
package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/raymondtc/clink-cli/pkg/models"
)

// Config holds the client configuration
type Config struct {
	AccessID      string
	AccessSecret  string
	EnterpriseID  string
	BaseURL       string
	Timeout       time.Duration
	EnableMock    bool
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		BaseURL:    "https://api.clink.cn",
		Timeout:    30 * time.Second,
		EnableMock: true,
	}
}

// Client is the HTTP client for Clink API
type Client struct {
	config     *Config
	httpClient *http.Client
}

// NewClient creates a new client
func NewClient(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// SetMock enables or disables mock mode
func (c *Client) SetMock(enable bool) {
	c.config.EnableMock = enable
}

// generateSignature creates HMAC-SHA256 signature
func (c *Client) generateSignature(params map[string]string) string {
	// Sort keys
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build param string
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, params[k]))
	}
	paramStr := strings.Join(parts, "")

	// Generate HMAC
	h := hmac.New(sha256.New, []byte(c.config.AccessSecret))
	h.Write([]byte(paramStr))
	return hex.EncodeToString(h.Sum(nil))
}

// Request makes an HTTP request
func (c *Client) Request(ctx context.Context, method, path string, params map[string]string, body io.Reader) (*models.APIResponse, error) {
	// Add common params
	if params == nil {
		params = make(map[string]string)
	}
	params["accessId"] = c.config.AccessID
	if c.config.EnterpriseID != "" {
		params["enterpriseId"] = c.config.EnterpriseID
	}
	params["timestamp"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	params["signature"] = c.generateSignature(params)

	// Build URL
	u, _ := url.Parse(c.config.BaseURL + path)
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	// Mock mode
	if c.config.EnableMock {
		return c.mockResponse(path, params), nil
	}

	// Make request
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result models.APIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// mockResponse returns mock data for testing
func (c *Client) mockResponse(path string, params map[string]string) *models.APIResponse {
	if strings.Contains(path, "callin") || strings.Contains(path, "inbound") {
		return &models.APIResponse{
			Code:    200,
			Message: "success",
			Data: models.ListResponse{
				Total: 2,
				List: []models.CallRecord{
					{
						ID:           "202401010001",
						Phone:        "13800138001",
						AgentID:      "1001",
						AgentName:    "张三",
						StartTime:    "2024-01-01 09:00:00",
						EndTime:      "2024-01-01 09:05:30",
						Duration:     330,
						Status:       "answered",
						RecordingURL: "https://example.com/recording1.mp3",
					},
					{
						ID:        "202401010002",
						Phone:     "13800138002",
						AgentID:   "1002",
						AgentName: "李四",
						StartTime: "2024-01-01 10:00:00",
						EndTime:   "2024-01-01 10:02:00",
						Duration:  120,
						Status:    "missed",
					},
				},
			},
		}
	}

	if strings.Contains(path, "callout") || strings.Contains(path, "outbound") {
		return &models.APIResponse{
			Code:    200,
			Message: "success",
			Data: models.ListResponse{
				Total: 1,
				List: []models.CallRecord{
					{
						ID:           "202401010003",
						Phone:        "13800138003",
						AgentID:      "1001",
						AgentName:    "张三",
						StartTime:    "2024-01-01 14:00:00",
						EndTime:      "2024-01-01 14:03:00",
						Duration:     180,
						Status:       "answered",
						RecordingURL: "https://example.com/recording3.mp3",
					},
				},
			},
		}
	}

	if strings.Contains(path, "agent") {
		return &models.APIResponse{
			Code:    200,
			Message: "success",
			Data: models.ListResponse{
				Total: 3,
				List: []models.Agent{
					{
						AgentID:   "1001",
						AgentName: "张三",
						Status:    "online",
						LoginTime: "2024-01-01 08:00:00",
					},
					{
						AgentID:     "1002",
						AgentName:   "李四",
						Status:      "busy",
						CurrentCall: "202401010004",
						LoginTime:   "2024-01-01 08:30:00",
					},
					{
						AgentID:   "1003",
						AgentName: "王五",
						Status:    "offline",
					},
				},
			},
		}
	}

	if strings.Contains(path, "queue") {
		return &models.APIResponse{
			Code:    200,
			Message: "success",
			Data: models.Queue{
				QueueID:      "Q001",
				QueueName:    "客服队列",
				WaitingCount: 5,
				AvgWaitTime:  120,
				AgentsOnline: 3,
				AgentsBusy:   2,
			},
		}
	}

	if strings.Contains(path, "dial") || strings.Contains(path, "call") {
		return &models.APIResponse{
			Code:    200,
			Message: "success",
			Data: models.CallResult{
				CallID: "202401010005",
				Status: "dialing",
				Phone:  params["phone"],
			},
		}
	}

	return &models.APIResponse{
		Code:    200,
		Message: "success",
		Data:    map[string]interface{}{},
	}
}
