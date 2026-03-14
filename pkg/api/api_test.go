package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/raymondtc/clink-cli/pkg/client"
)

func setupTestAPI() *API {
	config := client.DefaultConfig()
	c := client.NewClient(config)
	return NewAPI(c)
}

func TestGetInboundRecords(t *testing.T) {
	a := setupTestAPI()
	ctx := context.Background()
	
	records, total, err := a.GetInboundRecords(
		ctx,
		"2024-01-01 00:00:00",
		"2024-01-31 23:59:59",
		"",
		"",
		1,
		50,
	)
	
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, records, 2)
	assert.Equal(t, "202401010001", records[0].ID)
}

func TestGetOutboundRecords(t *testing.T) {
	a := setupTestAPI()
	ctx := context.Background()
	
	records, total, err := a.GetOutboundRecords(
		ctx,
		"2024-01-01 00:00:00",
		"2024-01-31 23:59:59",
		"",
		"",
		1,
		50,
	)
	
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, records, 1)
}

func TestGetAgentStatus(t *testing.T) {
	a := setupTestAPI()
	ctx := context.Background()
	
	agents, err := a.GetAgentStatus(ctx, "")
	
	assert.NoError(t, err)
	assert.Len(t, agents, 3)
	assert.Equal(t, "1001", agents[0].AgentID)
	assert.Equal(t, "online", agents[0].Status)
}

func TestMakeCall(t *testing.T) {
	a := setupTestAPI()
	ctx := context.Background()
	
	result, err := a.MakeCall(ctx, "13800138000", "1001", "")
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "dialing", result.Status)
}

func TestGetQueueStatus(t *testing.T) {
	a := setupTestAPI()
	ctx := context.Background()
	
	queue, err := a.GetQueueStatus(ctx, "")
	
	assert.NoError(t, err)
	assert.NotNil(t, queue)
	assert.Equal(t, "Q001", queue.QueueID)
	assert.Equal(t, 5, queue.WaitingCount)
}
