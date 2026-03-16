// Package api provides high-level API methods
package api

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/raymondtc/clink-cli/pkg/client"
	"github.com/raymondtc/clink-cli/pkg/models"
)

// API provides high-level API methods
type API struct {
	client *client.Client
}

// NewAPI creates a new API instance
func NewAPI(client *client.Client) *API {
	return &API{client: client}
}

// GetInboundRecords gets inbound call records
// API: GET /cc/list_cdr_ibs
func (a *API) GetInboundRecords(ctx context.Context, startTime, endTime, phone, agentID string, page, pageSize int) ([]models.CallRecord, int, error) {
	// 尝试解析不同格式的时间
	var start, end time.Time
	var err error
	
	// 尝试解析完整格式
	start, err = time.Parse("2006-01-02 15:04:05", startTime)
	if err != nil {
		// 尝试只解析日期，然后加上时间
		start, _ = time.Parse("2006-01-02", startTime)
		start = start.Add(0) // 00:00:00
	}
	
	end, err = time.Parse("2006-01-02 15:04:05", endTime)
	if err != nil {
		end, _ = time.Parse("2006-01-02", endTime)
		end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second) // 23:59:59
	}
	
	params := map[string]string{
		"startTime": strconv.FormatInt(start.UnixMilli(), 10),
		"endTime":   strconv.FormatInt(end.UnixMilli(), 10),
		"offset":    strconv.Itoa((page - 1) * pageSize),
		"limit":     strconv.Itoa(pageSize),
	}
	if phone != "" {
		params["customerNumber"] = phone
	}
	if agentID != "" {
		params["cno"] = agentID
	}

	resp, err := a.client.Request(ctx, "GET", "/cc/list_cdr_ibs", params, nil)
	if err != nil {
		return nil, 0, err
	}

	data, _ := json.Marshal(resp.Data)
	var listResp models.ListResponse
	if err := json.Unmarshal(data, &listResp); err != nil {
		return nil, 0, err
	}

	recordsData, _ := json.Marshal(listResp.List)
	var records []models.CallRecord
	if err := json.Unmarshal(recordsData, &records); err != nil {
		return nil, 0, err
	}

	return records, listResp.Total, nil
}

// GetOutboundRecords gets outbound call records
// API: GET /cc/list_cdr_obs
func (a *API) GetOutboundRecords(ctx context.Context, startTime, endTime, phone, agentID string, page, pageSize int) ([]models.CallRecord, int, error) {
	// 尝试解析不同格式的时间
	var start, end time.Time
	var err error
	
	// 尝试解析完整格式
	start, err = time.Parse("2006-01-02 15:04:05", startTime)
	if err != nil {
		// 尝试只解析日期，然后加上时间
		start, _ = time.Parse("2006-01-02", startTime)
		start = start.Add(0) // 00:00:00
	}
	
	end, err = time.Parse("2006-01-02 15:04:05", endTime)
	if err != nil {
		end, _ = time.Parse("2006-01-02", endTime)
		end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second) // 23:59:59
	}
	
	params := map[string]string{
		"startTime": strconv.FormatInt(start.UnixMilli(), 10),
		"endTime":   strconv.FormatInt(end.UnixMilli(), 10),
		"offset":    strconv.Itoa((page - 1) * pageSize),
		"limit":     strconv.Itoa(pageSize),
	}
	if phone != "" {
		params["customerNumber"] = phone
	}
	if agentID != "" {
		params["cno"] = agentID
	}

	resp, err := a.client.Request(ctx, "GET", "/cc/list_cdr_obs", params, nil)
	if err != nil {
		return nil, 0, err
	}

	data, _ := json.Marshal(resp.Data)
	var listResp models.ListResponse
	if err := json.Unmarshal(data, &listResp); err != nil {
		return nil, 0, err
	}

	recordsData, _ := json.Marshal(listResp.List)
	var records []models.CallRecord
	if err := json.Unmarshal(recordsData, &records); err != nil {
		return nil, 0, err
	}

	return records, listResp.Total, nil
}

// GetAgentStatus gets agent status
// API: GET /cc/agent_status
func (a *API) GetAgentStatus(ctx context.Context, agentID string) ([]models.Agent, error) {
	params := map[string]string{}
	if agentID != "" {
		params["cno"] = agentID
	}

	resp, err := a.client.Request(ctx, "GET", "/cc/agent_status", params, nil)
	if err != nil {
		return nil, err
	}

	// 天润API直接返回数组在agentStatus字段
	if resp.AgentStatus != nil {
		return resp.AgentStatus, nil
	}

	// 尝试从Data解析（兼容旧格式）
	data, _ := json.Marshal(resp.Data)
	var agents []models.Agent
	if err := json.Unmarshal(data, &agents); err != nil {
		return nil, err
	}

	return agents, nil
}

// MakeCall makes an outbound call
// API: POST /cc/callout
func (a *API) MakeCall(ctx context.Context, phone, agentID, displayNumber string) (*models.CallResult, error) {
	body := map[string]string{
		"customerNumber": phone,
		"cno":            agentID,
	}
	if displayNumber != "" {
		body["clid"] = displayNumber
	}

	bodyJSON, _ := json.Marshal(body)
	resp, err := a.client.Request(ctx, "POST", "/cc/callout", nil, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(resp.Data)
	var result models.CallResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetQueueStatus gets queue status
// API: GET /cc/queue_status
func (a *API) GetQueueStatus(ctx context.Context, queueID string) (*models.Queue, error) {
	params := map[string]string{}
	if queueID != "" {
		params["qno"] = queueID
	}

	resp, err := a.client.Request(ctx, "GET", "/cc/queue_status", params, nil)
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(resp.Data)
	var queue models.Queue
	if err := json.Unmarshal(data, &queue); err != nil {
		return nil, err
	}

	return &queue, nil
}

// Webcall makes a webcall (no agent required)
// API: POST /cc/webcall
func (a *API) Webcall(ctx context.Context, phone, displayNumber string) (*models.CallResult, error) {
	body := map[string]string{
		"customerNumber": phone,
	}
	if displayNumber != "" {
		body["clid"] = displayNumber
	}

	bodyJSON, _ := json.Marshal(body)
	resp, err := a.client.Request(ctx, "POST", "/cc/webcall", nil, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, err
	}

	// Webcall API 返回结构: {"result": {"requestUniqueId": "xxx"}, "requestId": "xxx"}
	if resp.Result != nil {
		data, _ := json.Marshal(resp.Result)
		var result struct {
			RequestUniqueID string `json:"requestUniqueId"`
		}
		if err := json.Unmarshal(data, &result); err == nil && result.RequestUniqueID != "" {
			return &models.CallResult{
				CallID: result.RequestUniqueID,
				Status: "submitted",
				Phone:  phone,
			}, nil
		}
	}

	return &models.CallResult{
		CallID: resp.RequestID,
		Status: "unknown",
		Phone:  phone,
	}, nil
}
