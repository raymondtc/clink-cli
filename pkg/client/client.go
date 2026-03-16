// Package client provides HTTP client for Clink API
package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/raymondtc/clink-cli/pkg/models"
)

// Config holds the client configuration
type Config struct {
	AccessID      string // AccessKeyId
	AccessSecret  string // AccessKeySecret
	EnterpriseID  string // 企业ID，某些API需要
	BaseURL       string // 如: https://api-sh.clink.cn
	Timeout       time.Duration
	EnableMock    bool
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		BaseURL:    "https://api-sh.clink.cn",
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

// generateSignature creates HMAC-SHA1 signature for Clink API
// 格式: GET + host + path + sorted query params
func (c *Client) generateSignature(method, host, path string, params map[string]string) string {
	// 1. 对参数名排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 构建查询字符串 (name=value&name=value...)
	var parts []string
	for _, k := range keys {
		// URL encode value
		encodedValue := url.QueryEscape(params[k])
		parts = append(parts, fmt.Sprintf("%s=%s", k, encodedValue))
	}
	queryString := strings.Join(parts, "&")

	// 3. 构建待签名字符串: METHOD + host + path + ? + queryString
	stringToSign := fmt.Sprintf("%s%s%s?%s", method, host, path, queryString)

	// 4. HMAC-SHA1
	h := hmac.New(sha1.New, []byte(c.config.AccessSecret))
	h.Write([]byte(stringToSign))

	// 5. Base64 + URL Encode
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return url.QueryEscape(signature)
}

// Request makes an HTTP request
func (c *Client) Request(ctx context.Context, method, path string, params map[string]string, body io.Reader) (*models.APIResponse, error) {
	if params == nil {
		params = make(map[string]string)
	}

	// 解析 BaseURL 获取 host
	u, _ := url.Parse(c.config.BaseURL)
	host := u.Host

	// 添加公共参数
	params["AccessKeyId"] = c.config.AccessID
	params["Expires"] = "60"
	// UTC 时间格式: yyyy-MM-ddTHH:mm:ssZ
	params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// 计算签名
	params["Signature"] = c.generateSignature(method, host, path, params)

	// 构建完整 URL
	fullURL := c.config.BaseURL + path + "?"
	var queryParts []string
	for k, v := range params {
		queryParts = append(queryParts, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}
	fullURL += strings.Join(queryParts, "&")

	// Mock mode
	if c.config.EnableMock {
		return c.mockResponse(path, params), nil
	}

	// Make request
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
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

	// Debug: print response
	fmt.Fprintf(os.Stderr, "Debug: API URL: %s\n", fullURL)
	fmt.Fprintf(os.Stderr, "Debug: Status: %d\n", resp.StatusCode)
	fmt.Fprintf(os.Stderr, "Debug: Response: %s\n", string(respBody[:min(len(respBody), 500)]))

	var result models.APIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// mockResponse returns mock data for testing
func (c *Client) mockResponse(path string, params map[string]string) *models.APIResponse {
	if strings.Contains(path, "list_cdr_ibs") {
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
				},
			},
		}
	}

	if strings.Contains(path, "list_cdr_obs") {
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

	if strings.Contains(path, "agent_status") {
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
				},
			},
		}
	}

	if strings.Contains(path, "callout") {
		return &models.APIResponse{
			Code:    200,
			Message: "success",
			Data: models.CallResult{
				CallID: "202401010005",
				Status: "dialing",
				Phone:  params["customerNumber"],
			},
		}
	}

	if strings.Contains(path, "queue_status") {
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

	return &models.APIResponse{
		Code:    200,
		Message: "success",
		Data:    map[string]interface{}{},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
