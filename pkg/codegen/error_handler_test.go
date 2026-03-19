package codegen

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestErrorHandler_HandleError(t *testing.T) {
	eh := NewErrorHandler([]ErrorConfig{
		{Code: 400, Message: "Bad Request", Action: "return"},
		{Code: 429, Message: "Rate Limited", Action: "retry"},
	})

	tests := []struct {
		name    string
		err     error
		wantErr bool
		check   func(error) bool
	}{
		{
			name:    "nil error",
			err:     nil,
			wantErr: false,
		},
		{
			name:    "regular error",
			err:     errors.New("network error"),
			wantErr: true,
			check: func(e error) bool {
				// Regular errors get wrapped in APIError
				apiErr, ok := e.(*APIError)
				return ok && apiErr.StatusCode == 0 && strings.Contains(apiErr.Message, "network error")
			},
		},
		{
			name:    "api error",
			err:     &APIError{StatusCode: 400, Message: "Bad Request"},
			wantErr: true,
			check: func(e error) bool {
				apiErr, ok := e.(*APIError)
				return ok && apiErr.StatusCode == 400
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := eh.HandleError(tt.err, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil && err != nil && !tt.check(err) {
				t.Errorf("HandleError() check failed")
			}
		})
	}
}

func TestErrorHandler_HandleResponseError(t *testing.T) {
	eh := NewErrorHandler([]ErrorConfig{
		{Code: 400, Message: "Bad Request", Action: "return"},
		{Code: 401, Message: "Unauthorized", Action: "return"},
		{Code: 429, Message: "Rate Limited", Action: "retry"},
		{Code: 500, Message: "Server Error", Action: "retry"},
	})

	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
		retryable  bool
	}{
		{"success 200", 200, false, false},
		{"success 201", 201, false, false},
		{"bad request", 400, true, false},
		{"unauthorized", 401, true, false},
		{"rate limited", 429, true, true},
		{"server error", 500, true, true},
		{"not found", 404, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := eh.HandleResponseError(tt.statusCode, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleResponseError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if apiErr, ok := err.(*APIError); ok {
				if apiErr.Retryable != tt.retryable {
					t.Errorf("HandleResponseError() retryable = %v, want %v", apiErr.Retryable, tt.retryable)
				}
			}
		})
	}
}

func TestErrorHandler_ShouldRetry(t *testing.T) {
	eh := NewErrorHandler([]ErrorConfig{
		{Code: 429, Message: "Rate Limited", Action: "retry"},
	})

	tests := []struct {
		name        string
		err         error
		attempt     int
		maxAttempts int
		want        bool
	}{
		{
			name:        "retryable error within limit",
			err:         &APIError{StatusCode: 429, Retryable: true},
			attempt:     1,
			maxAttempts: 3,
			want:        true,
		},
		{
			name:        "retryable error at limit",
			err:         &APIError{StatusCode: 429, Retryable: true},
			attempt:     3,
			maxAttempts: 3,
			want:        false,
		},
		{
			name:        "non-retryable error",
			err:         &APIError{StatusCode: 400, Retryable: false},
			attempt:     1,
			maxAttempts: 3,
			want:        false,
		},
		{
			name:        "network error",
			err:         errors.New("connection timeout"),
			attempt:     1,
			maxAttempts: 3,
			want:        true,
		},
		{
			name:        "nil error",
			err:         nil,
			attempt:     1,
			maxAttempts: 3,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eh.ShouldRetry(tt.err, tt.attempt, tt.maxAttempts)
			if got != tt.want {
				t.Errorf("ShouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetryWithBackoff_Execute(t *testing.T) {
	r := NewRetryWithBackoff(3, 10*time.Millisecond, 100*time.Millisecond)

	t.Run("success on first try", func(t *testing.T) {
		callCount := 0
		ctx := context.Background()
		err := r.Execute(ctx, func() error {
			callCount++
			return nil
		})
		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if callCount != 1 {
			t.Errorf("Execute() called %d times, want 1", callCount)
		}
	})

	t.Run("success after retries", func(t *testing.T) {
		callCount := 0
		ctx := context.Background()
		err := r.Execute(ctx, func() error {
			callCount++
			if callCount < 3 {
				return &APIError{StatusCode: 429, Retryable: true}
			}
			return nil
		})
		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if callCount != 3 {
			t.Errorf("Execute() called %d times, want 3", callCount)
		}
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		callCount := 0
		ctx := context.Background()
		err := r.Execute(ctx, func() error {
			callCount++
			return &APIError{StatusCode: 500, Retryable: true}
		})
		if err == nil {
			t.Error("Execute() expected error, got nil")
		}
		if callCount != 3 {
			t.Errorf("Execute() called %d times, want 3", callCount)
		}
	})

	t.Run("non-retryable error", func(t *testing.T) {
		callCount := 0
		ctx := context.Background()
		err := r.Execute(ctx, func() error {
			callCount++
			return &APIError{StatusCode: 400, Retryable: false}
		})
		if err == nil {
			t.Error("Execute() expected error, got nil")
		}
		if callCount != 1 {
			t.Errorf("Execute() called %d times, want 1", callCount)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := r.Execute(ctx, func() error {
			return nil
		})
		if err != context.Canceled {
			t.Errorf("Execute() error = %v, want %v", err, context.Canceled)
		}
	})
}

func TestRetryWithBackoff_ExecuteWithResult(t *testing.T) {
	r := NewRetryWithBackoff(3, 10*time.Millisecond, 100*time.Millisecond)

	t.Run("success with result", func(t *testing.T) {
		ctx := context.Background()
		result, err := r.ExecuteWithResult(ctx, func() (interface{}, error) {
			return "success", nil
		})
		if err != nil {
			t.Errorf("ExecuteWithResult() error = %v", err)
		}
		if result != "success" {
			t.Errorf("ExecuteWithResult() result = %v, want %v", result, "success")
		}
	})

	t.Run("error no result", func(t *testing.T) {
		ctx := context.Background()
		result, err := r.ExecuteWithResult(ctx, func() (interface{}, error) {
			return nil, errors.New("failed")
		})
		if err == nil {
			t.Error("ExecuteWithResult() expected error, got nil")
		}
		if result != nil {
			t.Errorf("ExecuteWithResult() result = %v, want nil", result)
		}
	})
}

func TestRetryWithBackoff_calculateDelay(t *testing.T) {
	r := NewRetryWithBackoff(5, 100*time.Millisecond, 5*time.Second)

	tests := []struct {
		name    string
		attempt int
		min     time.Duration
		max     time.Duration
	}{
		{"attempt 0", 0, 75 * time.Millisecond, 125 * time.Millisecond},
		{"attempt 1", 1, 150 * time.Millisecond, 250 * time.Millisecond},
		{"attempt 2", 2, 300 * time.Millisecond, 500 * time.Millisecond},
		{"attempt 3", 3, 600 * time.Millisecond, 1000 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := r.calculateDelay(tt.attempt)
			if delay < tt.min || delay > tt.max {
				t.Errorf("calculateDelay() = %v, want between %v and %v", delay, tt.min, tt.max)
			}
		})
	}
}

func TestGetRetryAfter(t *testing.T) {
	tests := []struct {
		name         string
		resp         *http.Response
		defaultDelay time.Duration
		minDelay     time.Duration
		maxDelay     time.Duration
	}{
		{
			name:         "nil response",
			resp:         nil,
			defaultDelay: time.Second,
			minDelay:     time.Second,
			maxDelay:     time.Second,
		},
		{
			name: "retry-after seconds",
			resp: &http.Response{
				Header: http.Header{"Retry-After": []string{"5"}},
			},
			defaultDelay: time.Second,
			minDelay:     5 * time.Second,
			maxDelay:     5 * time.Second,
		},
		{
			name: "rate limit reset",
			resp: &http.Response{
				Header: http.Header{"X-RateLimit-Reset": []string{"1705276800"}},
			},
			defaultDelay: time.Second,
			minDelay:     0,
			maxDelay:     time.Hour,
		},
		{
			name:         "no headers",
			resp:         &http.Response{Header: http.Header{}},
			defaultDelay: 2 * time.Second,
			minDelay:     2 * time.Second,
			maxDelay:     2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := GetRetryAfter(tt.resp, tt.defaultDelay)
			if delay < tt.minDelay || delay > tt.maxDelay {
				t.Errorf("GetRetryAfter() = %v, want between %v and %v", delay, tt.minDelay, tt.maxDelay)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		context string
		want    string
	}{
		{
			name:    "nil error",
			err:     nil,
			context: "test",
			want:    "",
		},
		{
			name:    "wrap error",
			err:     errors.New("original"),
			context: "context",
			want:    "context: original",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WrapError(tt.err, tt.context)
			if tt.want == "" {
				if got != nil {
					t.Errorf("WrapError() = %v, want nil", got)
				}
			} else if got.Error() != tt.want {
				t.Errorf("WrapError() = %v, want %v", got.Error(), tt.want)
			}
		})
	}
}

func TestErrorCheckerFunctions(t *testing.T) {
	t.Run("IsNotFound", func(t *testing.T) {
		if !IsNotFound(&APIError{StatusCode: 404}) {
			t.Error("IsNotFound(404) should be true")
		}
		if IsNotFound(&APIError{StatusCode: 400}) {
			t.Error("IsNotFound(400) should be false")
		}
		if IsNotFound(errors.New("error")) {
			t.Error("IsNotFound(generic error) should be false")
		}
	})

	t.Run("IsUnauthorized", func(t *testing.T) {
		if !IsUnauthorized(&APIError{StatusCode: 401}) {
			t.Error("IsUnauthorized(401) should be true")
		}
		if IsUnauthorized(&APIError{StatusCode: 403}) {
			t.Error("IsUnauthorized(403) should be false")
		}
	})

	t.Run("IsRateLimited", func(t *testing.T) {
		if !IsRateLimited(&APIError{StatusCode: 429}) {
			t.Error("IsRateLimited(429) should be true")
		}
		if IsRateLimited(&APIError{StatusCode: 500}) {
			t.Error("IsRateLimited(500) should be false")
		}
	})
}
