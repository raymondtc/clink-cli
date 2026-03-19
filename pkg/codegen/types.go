// Package codegen provides runtime support for generated code
package codegen

import (
	"context"
)

// RequestTransformConfig defines parameter transformation configuration
type RequestTransformConfig struct {
	Field    string `yaml:"field"`
	From     string `yaml:"from"`     // date, datetime, string, int
	To       string `yaml:"to"`       // timestamp, string, int
	Format   string `yaml:"format"`   // time format like "2006-01-02"
	Timezone string `yaml:"timezone"` // e.g., "Asia/Shanghai"
	EndOfDay bool   `yaml:"endOfDay"` // add 23:59:59 for end time
}

// ResponseMappingConfig defines response field mapping
type ResponseMappingConfig struct {
	From   string                 `yaml:"from"`
	To     string                 `yaml:"to"`
	Type   string                 `yaml:"type"`   // string, int, datetime, duration, enum
	Format string                 `yaml:"format"` // for datetime/duration
	Unit   string                 `yaml:"unit"`   // for duration: seconds, minutes
	Enum   map[interface{}]string `yaml:"enum,omitempty"`
}

// PaginationConfig defines pagination settings
type PaginationConfig struct {
	Enabled  bool `yaml:"enabled"`
	Request  struct {
		OffsetField string `yaml:"offsetField"`
		LimitField  string `yaml:"limitField"`
		PageField   string `yaml:"pageField,omitempty"`
	} `yaml:"request"`
	Response struct {
		TotalPath   string `yaml:"totalPath"`
		ItemsPath   string `yaml:"itemsPath"`
		HasMorePath string `yaml:"hasMorePath,omitempty"`
	} `yaml:"response"`
}

// ResponseConfig defines response handling configuration
type ResponseConfig struct {
	Type           string                  `yaml:"type"` // list, paged, single, simple
	Extract        string                  `yaml:"extract"`
	Pagination     PaginationConfig        `yaml:"pagination,omitempty"`
	Mapping        []ResponseMappingConfig `yaml:"mapping,omitempty"`
	SuccessMessage string                  `yaml:"successMessage,omitempty"`
	Output         OutputConfig            `yaml:"output,omitempty"`
}

// ErrorConfig defines error handling configuration
type ErrorConfig struct {
	Code       int    `yaml:"code"`
	Message    string `yaml:"message"`
	Action     string `yaml:"action"` // return, retry, ignore
	RetryAfter string `yaml:"retryAfter,omitempty"`
}

// OutputConfig defines output formatting configuration
type OutputConfig struct {
	Format    string   `yaml:"format"` // table, json, csv
	Columns   []string `yaml:"columns,omitempty"`
	SortBy    string   `yaml:"sortBy,omitempty"`
	SortOrder string   `yaml:"sortOrder,omitempty"` // asc, desc
}

// EndpointConfig defines the complete endpoint configuration
type EndpointConfig struct {
	OperationID string                   `yaml:"operationId"`
	Command     []string                 `yaml:"command"`
	Description string                   `yaml:"description"`
	Parameters  struct {
		UseTypes []string               `yaml:"useTypes,omitempty"`
		Fields   []ParameterConfig      `yaml:"fields,omitempty"`
	} `yaml:"parameters"`
	Request    struct {
		Transforms []RequestTransformConfig `yaml:"transforms,omitempty"`
	} `yaml:"request,omitempty"`
	Response   ResponseConfig           `yaml:"response"`
	Errors     []ErrorConfig            `yaml:"errors,omitempty"`
}

// ParameterConfig defines a single parameter configuration
type ParameterConfig struct {
	Name        string      `yaml:"name"`
	Flag        string      `yaml:"flag,omitempty"`
	Shorthand   string      `yaml:"shorthand,omitempty"`
	Type        string      `yaml:"type"`
	Description string      `yaml:"description,omitempty"`
	Default     interface{} `yaml:"default,omitempty"`
	Required    bool        `yaml:"required,omitempty"`
	Validate    string      `yaml:"validate,omitempty"` // phone, email, etc.
	Enum        map[interface{}]string `yaml:"enum,omitempty"`
}

// GeneratorConfig defines the full generator configuration
type GeneratorConfig struct {
	Version   string                    `yaml:"version"`
	Global    GlobalConfig              `yaml:"global"`
	Types     map[string]TypeDefinition `yaml:"types,omitempty"`
	Endpoints map[string]EndpointConfig `yaml:"endpoints"`
}

// GlobalConfig defines global settings
type GlobalConfig struct {
	OutputFormat    string `yaml:"outputFormat"`
	DefaultPageSize int    `yaml:"defaultPageSize"`
	TimeFormat      string `yaml:"timeFormat"`
	Client          struct {
		BaseURL string `yaml:"baseURL"`
		Timeout int    `yaml:"timeout"`
		Retry   struct {
			MaxAttempts int    `yaml:"maxAttempts"`
			Backoff     string `yaml:"backoff"`
		} `yaml:"retry"`
	} `yaml:"client"`
	Response struct {
		SuccessCode int    `yaml:"successCode"`
		CodePath    string `yaml:"codePath"`
		MessagePath string `yaml:"messagePath"`
		DataPath    string `yaml:"dataPath"`
	} `yaml:"response"`
}

// TypeDefinition defines a reusable type
type TypeDefinition struct {
	Fields map[string]ParameterConfig `yaml:"fields"`
}

// APIClient defines the interface for generated API clients
type APIClient interface {
	Do(ctx context.Context, method, path string, params, body interface{}) (interface{}, error)
}

// Renderable defines the interface for renderable data
type Renderable interface {
	ToTable(config *OutputConfig) (*Table, error)
}

// Table represents a formatted table
type Table struct {
	Headers []string
	Rows    []TableRow
}

// TableRow represents a table row
type TableRow struct {
	Cells []TableCell
}

// TableCell represents a table cell
type TableCell struct {
	Value string
	Color string // ANSI color code
	Align int    // -1 left, 0 default, 1 right
}
