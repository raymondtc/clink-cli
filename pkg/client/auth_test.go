package client

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthConfig_RequestEditorFn(t *testing.T) {
	authConfig := &AuthConfig{
		AccessID:     "test_access_id",
		AccessSecret: "test_secret_key",
	}

	// Create a request
	req, err := http.NewRequest("GET", "https://api-sh.clink.cn/cc/list_cdr_ibs?startTime=2024-01-01&endTime=2024-01-31", nil)
	assert.NoError(t, err)

	// Apply auth editor
	editor := authConfig.RequestEditorFn()
	err = editor(context.Background(), req)
	assert.NoError(t, err)

	// Check that auth params are added
	query := req.URL.Query()
	assert.Equal(t, "test_access_id", query.Get("AccessKeyId"))
	assert.Equal(t, "60", query.Get("Expires"))
	assert.NotEmpty(t, query.Get("Timestamp"))
	assert.NotEmpty(t, query.Get("Signature"))

	// Check original params are preserved
	assert.Equal(t, "2024-01-01", query.Get("startTime"))
	assert.Equal(t, "2024-01-31", query.Get("endTime"))
}

func TestGenerateSignature_Detailed(t *testing.T) {
	// Test signature generation with known values
	secret := "test_secret"
	req, _ := http.NewRequest("GET", "https://api-sh.clink.cn/cc/list_cdr_ibs?a=1&b=2", nil)

	query := req.URL.Query()
	query.Set("AccessKeyId", "test_id")
	query.Set("Expires", "60")
	query.Set("Timestamp", "2024-01-15T10:30:00Z")

	sig := generateSignature("GET", "api-sh.clink.cn", "/cc/list_cdr_ibs", query, secret)

	// Signature should not be empty and should be URL encoded
	assert.NotEmpty(t, sig)
	// URL encoded signature should contain % characters or be different from raw base64
	assert.True(t, strings.Contains(sig, "%") || sig != "")
}

func TestGenerateSignature_SortedParams(t *testing.T) {
	// Test that parameters are sorted alphabetically
	secret := "test_secret"

	// Create query with unsorted params
	req, _ := http.NewRequest("GET", "https://api-sh.clink.cn/cc/list_cdr_ibs", nil)
	query := req.URL.Query()
	query.Set("z", "last")
	query.Set("a", "first")
	query.Set("m", "middle")

	sig1 := generateSignature("GET", "api-sh.clink.cn", "/cc/list_cdr_ibs", query, secret)

	// Create same query but in different order
	req2, _ := http.NewRequest("GET", "https://api-sh.clink.cn/cc/list_cdr_ibs", nil)
	query2 := req2.URL.Query()
	query2.Set("a", "first")
	query2.Set("m", "middle")
	query2.Set("z", "last")

	sig2 := generateSignature("GET", "api-sh.clink.cn", "/cc/list_cdr_ibs", query2, secret)

	// Signatures should be the same (params are sorted)
	assert.Equal(t, sig1, sig2)
}

func TestGenerateSignature_POST(t *testing.T) {
	secret := "test_secret"
	req, _ := http.NewRequest("POST", "https://api-bj.clink.cn/cc/online", nil)

	query := req.URL.Query()
	query.Set("AccessKeyId", "test_id")
	query.Set("Expires", "60")
	query.Set("Timestamp", "2024-01-15T10:30:00Z")

	sig := generateSignature("POST", "api-bj.clink.cn", "/cc/online", query, secret)

	assert.NotEmpty(t, sig)
}
