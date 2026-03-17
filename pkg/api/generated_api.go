// Package api provides high-level API methods using generated code
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/raymondtc/clink-cli/pkg/client"
	"github.com/raymondtc/clink-cli/pkg/generated"
)

// GeneratedAPI provides high-level API methods using generated client
type GeneratedAPI struct {
	client *generated.ClientWithResponses
	config *client.AuthConfig
}

// NewGeneratedAPI creates a new API instance using generated client
func NewGeneratedAPI(baseURL string, config *client.AuthConfig) (*GeneratedAPI, error) {
	// Create HTTP client with authentication
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create client with authentication editor
	c, err := generated.NewClientWithResponses(
		baseURL,
		generated.WithHTTPClient(httpClient),
		generated.WithRequestEditorFn(config.RequestEditorFn()),
	)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	return &GeneratedAPI{
		client: c,
		config: config,
	}, nil
}

// GetInboundRecords gets inbound call records
func (a *GeneratedAPI) GetInboundRecords(ctx context.Context, startTime, endTime, phone, agentID string, page, pageSize int) ([]generated.InboundCallRecord, int, error) {
	// Parse time and convert to Unix seconds (API expects seconds, not milliseconds)
	start, _ := time.Parse("2006-01-02", startTime)
	end, _ := time.Parse("2006-01-02", endTime)
	// Set end time to end of day
	end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	offset := (page - 1) * pageSize
	params := &generated.ListCdrIbsParams{
		StartTime: start.Unix(),
		EndTime:   end.Unix(),
		Offset:    &offset,
		Limit:     &pageSize,
	}
	if phone != "" {
		params.CustomerNumber = &phone
	}
	if agentID != "" {
		params.Cno = &agentID
	}

	resp, err := a.client.ListCdrIbsWithResponse(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list cdr ibs: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, 0, fmt.Errorf("unexpected response status: %d, body: %s", resp.StatusCode(), string(resp.Body))
	}

	if resp.JSON200.CdrIb == nil {
		return []generated.InboundCallRecord{}, 0, nil
	}

	total := 0
	if resp.JSON200.TotalCount != nil {
		total = *resp.JSON200.TotalCount
	}

	return *resp.JSON200.CdrIb, total, nil
}

// GetOutboundRecords gets outbound call records
func (a *GeneratedAPI) GetOutboundRecords(ctx context.Context, startTime, endTime, phone, agentID string, page, pageSize int) ([]generated.OutboundCallRecord, int, error) {
	// Parse time and convert to Unix seconds (API expects seconds, not milliseconds)
	start, _ := time.Parse("2006-01-02", startTime)
	end, _ := time.Parse("2006-01-02", endTime)
	// Set end time to end of day
	end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	offset := (page - 1) * pageSize
	params := &generated.ListCdrObsParams{
		StartTime: start.Unix(),
		EndTime:   end.Unix(),
		Offset:    &offset,
		Limit:     &pageSize,
	}
	if phone != "" {
		params.CustomerNumber = &phone
	}
	if agentID != "" {
		params.Cno = &agentID
	}

	resp, err := a.client.ListCdrObsWithResponse(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list cdr obs: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, 0, fmt.Errorf("unexpected response status: %d, body: %s", resp.StatusCode(), string(resp.Body))
	}

	if resp.JSON200.CdrObs == nil {
		return []generated.OutboundCallRecord{}, 0, nil
	}

	total := 0
	if resp.JSON200.TotalCount != nil {
		total = *resp.JSON200.TotalCount
	}

	return *resp.JSON200.CdrObs, total, nil
}

// GetAgentStatus gets agent status
func (a *GeneratedAPI) GetAgentStatus(ctx context.Context, agentID string) ([]generated.AgentStatus, error) {
	var params *generated.ListAgentStatusParams
	if agentID != "" {
		params = &generated.ListAgentStatusParams{
			Cno: &agentID,
		}
	}

	resp, err := a.client.ListAgentStatusWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list agent status: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response status: %d", resp.StatusCode())
	}

	if resp.JSON200.AgentStatus == nil {
		return []generated.AgentStatus{}, nil
	}

	return *resp.JSON200.AgentStatus, nil
}

// MakeCall makes an outbound call
func (a *GeneratedAPI) MakeCall(ctx context.Context, phone, agentID, displayNumber string) (*generated.CallResult, error) {
	body := generated.CalloutJSONRequestBody{
		Cno:            agentID,
		CustomerNumber: phone,
	}
	if displayNumber != "" {
		body.Clid = &displayNumber
	}

	resp, err := a.client.CalloutWithResponse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("callout: %w", err)
	}

	if resp.JSON200 == nil || resp.JSON200.Data == nil {
		return nil, fmt.Errorf("unexpected response status: %d", resp.StatusCode())
	}

	return resp.JSON200.Data, nil
}

// Webcall makes a webcall (no agent required)
func (a *GeneratedAPI) Webcall(ctx context.Context, phone, displayNumber, ivrName string, extraParams map[string]string) (*generated.CallResult, error) {
	body := generated.WebcallJSONRequestBody{
		CustomerNumber: phone,
	}
	if displayNumber != "" {
		body.Clid = &displayNumber
	}
	if ivrName != "" {
		body.IvrName = &ivrName
	}
	// Add extra parameters
	if len(extraParams) > 0 {
		vars := make(map[string]interface{})
		for k, v := range extraParams {
			vars[k] = v
		}
		body.Variables = &vars
	}

	resp, err := a.client.WebcallWithResponse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("webcall: %w", err)
	}

	if resp.JSON200 == nil || resp.JSON200.Result == nil {
		return nil, fmt.Errorf("unexpected response status: %d", resp.StatusCode())
	}

	result := &generated.CallResult{}
	if resp.JSON200.Result.RequestUniqueId != nil {
		result.CallId = resp.JSON200.Result.RequestUniqueId
	}
	result.CustomerNumber = &phone
	result.Status = strPtr("submitted")
	return result, nil
}

func strPtr(s string) *string {
	return &s
}

// Online logs in an agent
func (a *GeneratedAPI) Online(ctx context.Context, agentID, qno, bindTel string, bindType int) error {
	body := generated.OnlineJSONRequestBody{
		Cno: agentID,
	}
	if qno != "" {
		body.Qno = &qno
	}
	if bindTel != "" {
		body.BindTel = &bindTel
		body.BindType = &bindType
	}

	resp, err := a.client.OnlineWithResponse(ctx, body)
	if err != nil {
		return fmt.Errorf("online: %w", err)
	}

	if resp.JSON200 == nil {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode())
	}

	return nil
}

// Offline logs out an agent
func (a *GeneratedAPI) Offline(ctx context.Context, agentID string) error {
	body := generated.OfflineJSONRequestBody{
		Cno: agentID,
	}

	resp, err := a.client.OfflineWithResponse(ctx, body)
	if err != nil {
		return fmt.Errorf("offline: %w", err)
	}

	if resp.JSON200 == nil {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode())
	}

	return nil
}

// Pause sets agent to busy status
func (a *GeneratedAPI) Pause(ctx context.Context, agentID string, pauseType int, description string) error {
	body := generated.PauseJSONRequestBody{
		Cno: agentID,
	}
	if pauseType > 0 {
		body.PauseType = &pauseType
	}
	if description != "" {
		body.Description = &description
	}

	resp, err := a.client.PauseWithResponse(ctx, body)
	if err != nil {
		return fmt.Errorf("pause: %w", err)
	}

	if resp.JSON200 == nil {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode())
	}

	return nil
}

// Unpause sets agent to ready status
func (a *GeneratedAPI) Unpause(ctx context.Context, agentID string) error {
	body := generated.UnpauseJSONRequestBody{
		Cno: agentID,
	}

	resp, err := a.client.UnpauseWithResponse(ctx, body)
	if err != nil {
		return fmt.Errorf("unpause: %w", err)
	}

	if resp.JSON200 == nil {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode())
	}

	return nil
}

// Hangup hangs up the current call
func (a *GeneratedAPI) Hangup(ctx context.Context, agentID string) error {
	body := generated.UnlinkJSONRequestBody{
		Cno: agentID,
	}

	resp, err := a.client.UnlinkWithResponse(ctx, body)
	if err != nil {
		return fmt.Errorf("hangup: %w", err)
	}

	if resp.JSON200 == nil {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode())
	}

	return nil
}

// Hold puts the current call on hold
func (a *GeneratedAPI) Hold(ctx context.Context, agentID string) error {
	body := generated.HoldJSONRequestBody{
		Cno: agentID,
	}

	resp, err := a.client.HoldWithResponse(ctx, body)
	if err != nil {
		return fmt.Errorf("hold: %w", err)
	}

	if resp.JSON200 == nil {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode())
	}

	return nil
}

// Unhold resumes the held call
func (a *GeneratedAPI) Unhold(ctx context.Context, agentID string) error {
	body := generated.UnholdJSONRequestBody{
		Cno: agentID,
	}

	resp, err := a.client.UnholdWithResponse(ctx, body)
	if err != nil {
		return fmt.Errorf("unhold: %w", err)
	}

	if resp.JSON200 == nil {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode())
	}

	return nil
}

// GetQueueStatus gets queue status
func (a *GeneratedAPI) GetQueueStatus(ctx context.Context, queueID string) (*generated.QueueStatus, error) {
	var params *generated.GetQueueStatusParams
	if queueID != "" {
		params = &generated.GetQueueStatusParams{
			Qnos: &queueID,
		}
	}

	resp, err := a.client.GetQueueStatusWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("get queue status: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response status: %d, body: %s", resp.StatusCode(), string(resp.Body))
	}
	if resp.JSON200.QueueStatus == nil || len(*resp.JSON200.QueueStatus) == 0 {
		return nil, fmt.Errorf("no queue status data available")
	}

	return &(*resp.JSON200.QueueStatus)[0], nil
}

// ListQueues gets queue list
func (a *GeneratedAPI) ListQueues(ctx context.Context, offset, limit int) ([]generated.Queue, error) {
	params := &generated.ListQueuesParams{
		Offset: &offset,
		Limit:  &limit,
	}

	resp, err := a.client.ListQueuesWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list queues: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response status: %d", resp.StatusCode())
	}

	if resp.JSON200.Queues == nil {
		return []generated.Queue{}, nil
	}

	return *resp.JSON200.Queues, nil
}

// Transfer transfers a call
func (a *GeneratedAPI) Transfer(ctx context.Context, agentID string, transferType int, transferObject string) error {
	body := generated.TransferJSONRequestBody{
		Cno:            agentID,
		TransferType:   transferType,
		TransferObject: transferObject,
	}

	resp, err := a.client.TransferWithResponse(ctx, body)
	if err != nil {
		return fmt.Errorf("transfer: %w", err)
	}

	if resp.JSON200 == nil {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode())
	}

	return nil
}
