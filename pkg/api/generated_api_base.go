// Package api provides high-level API methods using generated code
package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/raymondtc/clink-cli/pkg/client"
	"github.com/raymondtc/clink-cli/pkg/codegen"
	"github.com/raymondtc/clink-cli/pkg/generated"
)

// GeneratedAPI provides high-level API methods using generated client
type GeneratedAPI struct {
	client *generated.ClientWithResponses
	config *client.AuthConfig
	rb     *codegen.RequestBuilder
	rp     *codegen.ResponseParser
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

	rb, err := codegen.NewRequestBuilder("Asia/Shanghai", 10)
	if err != nil {
		return nil, fmt.Errorf("create request builder: %w", err)
	}

	rp, err := codegen.NewResponseParser("Asia/Shanghai")
	if err != nil {
		return nil, fmt.Errorf("create response parser: %w", err)
	}

	return &GeneratedAPI{
		client: c,
		config: config,
		rb:     rb,
		rp:     rp,
	}, nil
}
