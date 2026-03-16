// Package api provides high-level API methods
package api

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

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
	params := map[string]string{
		"startTime": startTime,
		"endTime":   endTime,
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
	params := map[string]string{
		"startTime": startTime,
		"endTime":   endTime,
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

	data, _ := json.Marshal(resp.Data)
	var listResp models.ListResponse
	if err := json.Unmarshal(data, &listResp); err != nil {
		return nil, err
	}

	agentsData, _ := json.Marshal(listResp.List)
	var agents []models.Agent
	if err := json.Unmarshal(agentsData, &agents); err != nil {
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
