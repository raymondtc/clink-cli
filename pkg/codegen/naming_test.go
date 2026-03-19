package codegen

import (
	"testing"
)

func TestToValidIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"simple", "name", "name"},
		{"hyphen", "request-id", "requestId"},
		{"multiple hyphens", "customer-phone-number", "customerPhoneNumber"},
		{"underscore", "bind_type", "bindType"},
		{"mixed", "request_unique_id", "requestUniqueId"},
		{"space", "customer number", "customerNumber"},
		{"keyword type", "type", "typeVal"},
		{"keyword range", "range", "rangeVal"},
		{"keyword map", "map", "mapVal"},
		{"uppercase", "RequestID", "requestID"},
		{"mixed case", "CustomerNumber", "customerNumber"},
		{"camelCase", "customerNumber", "customerNumber"},
		{"camelCase multi", "bindType", "bindType"},
		{"with spaces", "customer number", "customerNumber"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToValidIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("ToValidIdentifier(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"simple", "name", "Name"},
		{"hyphen", "request-id", "RequestId"},
		{"multiple hyphens", "customer-phone-number", "CustomerPhoneNumber"},
		{"underscore", "bind_type", "BindType"},
		{"mixed", "request_unique_id", "RequestUniqueId"},
		{"uppercase", "RequestID", "RequestID"},
		{"mixed case", "CustomerNumber", "CustomerNumber"},
		{"camelCase", "customerNumber", "CustomerNumber"},
		{"camelCase multi", "bindType", "BindType"},
		{"request_id", "requestId", "RequestId"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToPascalCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty", "", false},
		{"simple", "name", true},
		{"with underscore", "bind_type", true},
		{"starts with digit", "1name", false},
		{"with hyphen", "request-id", false},
		{"with space", "customer number", false},
		{"keyword", "type", false},
		{"uppercase", "Name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidIdentifier(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
