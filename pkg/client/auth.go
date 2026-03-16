// Package client provides HTTP client for Clink API with authentication
package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	AccessID     string
	AccessSecret string
}

// RequestEditorFn creates a request editor function that adds Clink API authentication
func (c *AuthConfig) RequestEditorFn() func(ctx context.Context, req *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		// Parse existing query parameters
		query := req.URL.Query()

		// Add authentication parameters
		query.Set("AccessKeyId", c.AccessID)
		query.Set("Expires", "60")
		query.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))

		// Calculate signature (signature is already URL encoded in generateSignature)
		signature := generateSignature(req.Method, req.URL.Host, req.URL.Path, query, c.AccessSecret)
		// Note: query.Set will URL encode the value again, so we need to set raw value
		query.Set("Signature", "PLACEHOLDER")

		// Build final query
		finalQuery := query.Encode()
		// Replace placeholder with actual signature
		finalQuery = strings.Replace(finalQuery, "Signature=PLACEHOLDER", "Signature="+signature, 1)

		req.URL.RawQuery = finalQuery
		return nil
	}
}

// generateSignature creates HMAC-SHA1 signature for Clink API
// Format: METHOD + host + path + ? + sorted query params
func generateSignature(method, host, path string, query url.Values, secret string) string {
	// Get all parameter keys and sort them
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build query string: name=value&name=value...
	var parts []string
	for _, k := range keys {
		// URL encode value
		encodedValue := url.QueryEscape(query.Get(k))
		parts = append(parts, fmt.Sprintf("%s=%s", k, encodedValue))
	}
	queryString := ""
	if len(parts) > 0 {
		queryString = parts[0]
		for i := 1; i < len(parts); i++ {
			queryString += "&" + parts[i]
		}
	}

	// Build string to sign: METHOD + host + path + ? + queryString
	stringToSign := fmt.Sprintf("%s%s%s?%s", method, host, path, queryString)
	// HMAC-SHA1
	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(stringToSign))

	// Base64 + URL Encode
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return url.QueryEscape(signature)
}
