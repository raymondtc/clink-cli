// Package response provides common response handling utilities
package response

import (
	"encoding/json"
	"fmt"
)

// ResponseHandler handles API responses uniformly
type ResponseHandler struct {
	Operation string
}

// NewHandler creates a new response handler
func NewHandler(operation string) *ResponseHandler {
	return &ResponseHandler{Operation: operation}
}

// HandleResult handles a result response
func (h *ResponseHandler) HandleResult(data interface{}, err error) error {
	if err != nil {
		return fmt.Errorf("%s failed: %w", h.Operation, err)
	}
	if data == nil {
		return fmt.Errorf("%s returned empty result", h.Operation)
	}
	return nil
}

// HandleList handles a list response
func (h *ResponseHandler) HandleList(data interface{}, total int, err error) ([]interface{}, int, error) {
	if err != nil {
		return nil, 0, fmt.Errorf("%s failed: %w", h.Operation, err)
	}
	if data == nil {
		return []interface{}{}, 0, nil
	}
	
	// Convert to []interface{} if needed
	switch v := data.(type) {
	case []interface{}:
		return v, total, nil
	default:
		// Try to marshal and unmarshal
		bytes, err := json.Marshal(data)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal response: %w", err)
		}
		var result []interface{}
		if err := json.Unmarshal(bytes, &result); err != nil {
			return nil, 0, fmt.Errorf("unmarshal response: %w", err)
		}
		return result, total, nil
	}
}

// Handler is a shortcut to create handler and handle result
func Handler(operation string, data interface{}, err error) error {
	return NewHandler(operation).HandleResult(data, err)
}

// ListHandler handles list responses with typed data
type ListHandler[T any] struct {
	Operation string
}

// NewListHandler creates a new typed list handler
func NewListHandler[T any](operation string) *ListHandler[T] {
	return &ListHandler[T]{Operation: operation}
}

// Handle processes the list response
func (h *ListHandler[T]) Handle(items *[]T, total *int, httpStatus int, body []byte, err error) ([]T, int, error) {
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", h.Operation, err)
	}
	
	if items == nil {
		return []T{}, 0, nil
	}
	
	t := 0
	if total != nil {
		t = *total
	}
	
	return *items, t, nil
}

// Error represents a structured error
type Error struct {
	Operation string
	Message   string
	Cause     error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Operation, e.Message)
}

// Wrap wraps an error with context
func Wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{
		Operation: operation,
		Message:   "failed",
		Cause:     err,
	}
}

// Wrapf wraps an error with formatted message
func Wrapf(operation string, format string, args ...interface{}) error {
	return &Error{
		Operation: operation,
		Message:   fmt.Sprintf(format, args...),
	}
}

// MustBeOk checks if response status is OK
func MustBeOk(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}
	return fmt.Errorf("unexpected status: %d, body: %s", statusCode, string(body))
}

// ExtractResult extracts result from response body
func ExtractResult(body []byte, target interface{}) error {
	return json.Unmarshal(body, target)
}
