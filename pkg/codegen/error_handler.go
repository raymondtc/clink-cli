package codegen

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// APIError represents an API error
type APIError struct {
	StatusCode int
	Code       int
	Message    string
	Retryable  bool
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("API error [%d/%d]: %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("API error [%d]: %s", e.StatusCode, e.Message)
}

// ErrorHandler handles API errors and retry logic
type ErrorHandler struct {
	ErrorConfigs []ErrorConfig
}

// NewErrorHandler creates a new ErrorHandler
func NewErrorHandler(configs []ErrorConfig) *ErrorHandler {
	return &ErrorHandler{
		ErrorConfigs: configs,
	}
}

// HandleError processes an error and returns an appropriate APIError
func (eh *ErrorHandler) HandleError(err error, resp *http.Response) error {
	if err == nil {
		return nil
	}

	// If it's already an APIError, return it
	if apiErr, ok := err.(*APIError); ok {
		return apiErr
	}

	// Create APIError from response if available
	if resp != nil {
		return eh.createErrorFromResponse(resp)
	}

	// Network or other errors
	return &APIError{
		StatusCode: 0,
		Message:    err.Error(),
		Retryable:  eh.isRetryableError(err),
	}
}

// HandleResponseError checks if a response indicates an error
func (eh *ErrorHandler) HandleResponseError(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Find error config for this status code
	for _, config := range eh.ErrorConfigs {
		if config.Code == statusCode {
			return &APIError{
				StatusCode: statusCode,
				Code:       config.Code,
				Message:    config.Message,
				Retryable:  config.Action == "retry",
			}
		}
	}

	// Default error handling based on status code
	return &APIError{
		StatusCode: statusCode,
		Message:    eh.defaultErrorMessage(statusCode),
		Retryable:  eh.isRetryableStatus(statusCode),
	}
}

// createErrorFromResponse creates an APIError from HTTP response
func (eh *ErrorHandler) createErrorFromResponse(resp *http.Response) *APIError {
	statusCode := resp.StatusCode

	// Try to extract error code from response headers
	code := 0
	if codeStr := resp.Header.Get("X-Error-Code"); codeStr != "" {
		code, _ = strconv.Atoi(codeStr)
	}

	// Find matching error config
	for _, config := range eh.ErrorConfigs {
		if config.Code == statusCode || config.Code == code {
			return &APIError{
				StatusCode: statusCode,
				Code:       code,
				Message:    config.Message,
				Retryable:  config.Action == "retry",
			}
		}
	}

	return &APIError{
		StatusCode: statusCode,
		Code:       code,
		Message:    eh.defaultErrorMessage(statusCode),
		Retryable:  eh.isRetryableStatus(statusCode),
	}
}

// defaultErrorMessage returns a default error message for a status code
func (eh *ErrorHandler) defaultErrorMessage(statusCode int) string {
	switch statusCode {
	case 400:
		return "Bad request"
	case 401:
		return "Unauthorized - please check your credentials"
	case 403:
		return "Forbidden - you don't have permission"
	case 404:
		return "Resource not found"
	case 429:
		return "Too many requests - please slow down"
	case 500:
		return "Internal server error"
	case 502:
		return "Bad gateway"
	case 503:
		return "Service unavailable"
	case 504:
		return "Gateway timeout"
	default:
		return fmt.Sprintf("HTTP error %d", statusCode)
	}
}

// isRetryableStatus checks if a status code indicates a retryable error
func (eh *ErrorHandler) isRetryableStatus(statusCode int) bool {
	// 5xx errors and 429 (rate limit) are retryable
	return statusCode >= 500 || statusCode == 429
}

// isRetryableError checks if an error is retryable
func (eh *ErrorHandler) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"no such host",
		"timeout",
		"temporary failure",
		"i/o timeout",
	}

	for _, pattern := range retryableErrors {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// ShouldRetry determines if an operation should be retried
func (eh *ErrorHandler) ShouldRetry(err error, attempt int, maxAttempts int) bool {
	if attempt >= maxAttempts {
		return false
	}

	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Retryable
	}

	return eh.isRetryableError(err)
}

// RetryWithBackoff performs a retryable operation with exponential backoff
type RetryWithBackoff struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// NewRetryWithBackoff creates a new RetryWithBackoff
func NewRetryWithBackoff(maxAttempts int, baseDelay, maxDelay time.Duration) *RetryWithBackoff {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	if baseDelay <= 0 {
		baseDelay = 100 * time.Millisecond
	}
	if maxDelay <= 0 {
		maxDelay = 30 * time.Second
	}

	return &RetryWithBackoff{
		MaxAttempts: maxAttempts,
		BaseDelay:   baseDelay,
		MaxDelay:    maxDelay,
	}
}

// Execute executes an operation with retry logic
func (r *RetryWithBackoff) Execute(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt < r.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		eh := NewErrorHandler(nil)
		if !eh.ShouldRetry(err, attempt, r.MaxAttempts) {
			return err
		}

		// Calculate backoff delay
		delay := r.calculateDelay(attempt)

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// ExecuteWithResult executes an operation that returns a result with retry logic
func (r *RetryWithBackoff) ExecuteWithResult(ctx context.Context, operation func() (interface{}, error)) (interface{}, error) {
	var lastErr error

	for attempt := 0; attempt < r.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		result, err := operation()
		if err == nil {
			return result, nil
		}

		lastErr = err

		eh := NewErrorHandler(nil)
		if !eh.ShouldRetry(err, attempt, r.MaxAttempts) {
			return nil, err
		}

		delay := r.calculateDelay(attempt)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Continue
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// calculateDelay calculates the delay for a given attempt using exponential backoff
func (r *RetryWithBackoff) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: baseDelay * 2^attempt
	delay := r.BaseDelay * time.Duration(1<<attempt)

	// Add jitter (±25%)
	jitter := delay / 4
	delay = delay + time.Duration(float64(jitter)*(0.5-float64(attempt%10)/10))

	// Cap at maxDelay
	if delay > r.MaxDelay {
		delay = r.MaxDelay
	}

	return delay
}

// GetRetryAfter extracts retry-after duration from response
func GetRetryAfter(resp *http.Response, defaultDelay time.Duration) time.Duration {
	if resp == nil {
		return defaultDelay
	}

	// Try Retry-After header (seconds or HTTP date)
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		// Try parsing as seconds
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			return time.Duration(seconds) * time.Second
		}

		// Try parsing as HTTP date
		if t, err := http.ParseTime(retryAfter); err == nil {
			return time.Until(t)
		}
	}

	// Try X-RateLimit-Reset header (timestamp)
	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		if timestamp, err := strconv.ParseInt(reset, 10, 64); err == nil {
			resetTime := time.Unix(timestamp, 0)
			delay := time.Until(resetTime)
			if delay > 0 {
				return delay
			}
		}
	}

	return defaultDelay
}

// WrapError wraps an error with additional context
func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

// IsNotFound checks if an error is a "not found" error
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 404
	}
	return false
}

// IsUnauthorized checks if an error is an "unauthorized" error
func IsUnauthorized(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 401
	}
	return false
}

// IsRateLimited checks if an error is a rate limit error
func IsRateLimited(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 429
	}
	return false
}
