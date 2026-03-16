package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "https://api-sh.clink.cn", config.BaseURL)
	assert.True(t, config.EnableMock)
}

func TestNewClient(t *testing.T) {
	config := DefaultConfig()
	client := NewClient(config)
	assert.NotNil(t, client)
	assert.Equal(t, config, client.config)
}

func TestClientMockMode(t *testing.T) {
	config := DefaultConfig()
	config.EnableMock = true
	client := NewClient(config)
	
	// Test mock response for inbound records
	resp, err := client.Request(context.Background(), "GET", "/api/callin/records", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.Code)
	assert.NotNil(t, resp.Data)
}

func TestGenerateSignature(t *testing.T) {
	config := &Config{
		AccessSecret: "test_secret",
	}
	client := NewClient(config)
	
	params := map[string]string{
		"a": "1",
		"b": "2",
	}
	
	sig := client.generateSignature("GET", "api-sh.clink.cn", "/cc/list_cdr_ibs", params)
	assert.NotEmpty(t, sig)
}
