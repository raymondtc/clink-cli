package codegen

import (
	"context"
	"testing"
)

func TestRequestBuilder_BuildParams(t *testing.T) {
	rb, err := NewRequestBuilder("Asia/Shanghai", 10)
	if err != nil {
		t.Fatalf("NewRequestBuilder failed: %v", err)
	}

	tests := []struct {
		name    string
		flags   map[string]interface{}
		config  EndpointConfig
		wantErr bool
	}{
		{
			name: "simple params",
			flags: map[string]interface{}{
				"phone": "13800138000",
				"agent": "1001",
			},
			config: EndpointConfig{
				Parameters: struct {
					UseTypes []string          `yaml:"useTypes,omitempty"`
					Fields   []ParameterConfig `yaml:"fields,omitempty"`
				}{
					Fields: []ParameterConfig{
						{Name: "customerNumber", Flag: "phone", Type: "string"},
						{Name: "cno", Flag: "agent", Type: "string"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing required param",
			flags: map[string]interface{}{
				"phone": "",
			},
			config: EndpointConfig{
				Parameters: struct {
					UseTypes []string          `yaml:"useTypes,omitempty"`
					Fields   []ParameterConfig `yaml:"fields,omitempty"`
				}{
					Fields: []ParameterConfig{
						{Name: "customerNumber", Flag: "phone", Type: "string", Required: true},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := rb.BuildParams(ctx, tt.flags, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequestBuilder_validateValue(t *testing.T) {
	rb, err := NewRequestBuilder("Asia/Shanghai", 10)
	if err != nil {
		t.Fatalf("NewRequestBuilder failed: %v", err)
	}

	tests := []struct {
		name    string
		value   interface{}
		field   ParameterConfig
		wantErr bool
	}{
		{
			name:    "valid string",
			value:   "test",
			field:   ParameterConfig{Name: "test", Type: "string"},
			wantErr: false,
		},
		{
			name:    "valid int",
			value:   42,
			field:   ParameterConfig{Name: "test", Type: "int"},
			wantErr: false,
		},
		{
			name:    "valid phone",
			value:   "13800138000",
			field:   ParameterConfig{Name: "phone", Type: "string", Validate: "phone"},
			wantErr: false,
		},
		{
			name:    "invalid phone",
			value:   "123",
			field:   ParameterConfig{Name: "phone", Type: "string", Validate: "phone"},
			wantErr: true,
		},
		{
			name:    "valid date",
			value:   "2024-01-15",
			field:   ParameterConfig{Name: "date", Type: "date"},
			wantErr: false,
		},
		{
			name:    "invalid date",
			value:   "15-01-2024",
			field:   ParameterConfig{Name: "date", Type: "date"},
			wantErr: true,
		},
		{
			name:    "valid bool",
			value:   true,
			field:   ParameterConfig{Name: "test", Type: "bool"},
			wantErr: false,
		},
		{
			name:    "invalid type for string field",
			value:   123,
			field:   ParameterConfig{Name: "test", Type: "string"},
			wantErr: false, // Will be converted to string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rb.validateValue(tt.value, tt.field)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequestBuilder_applyTransforms(t *testing.T) {
	rb, err := NewRequestBuilder("Asia/Shanghai", 10)
	if err != nil {
		t.Fatalf("NewRequestBuilder failed: %v", err)
	}

	tests := []struct {
		name       string
		params     map[string]interface{}
		transforms []RequestTransformConfig
		wantErr    bool
		check      func(map[string]interface{}) bool
	}{
		{
			name: "date to timestamp",
			params: map[string]interface{}{
				"startTime": "2024-01-15",
			},
			transforms: []RequestTransformConfig{
				{
					Field:  "startTime",
					From:   "date",
					To:     "timestamp",
					Format: "2006-01-02",
				},
			},
			wantErr: false,
			check: func(params map[string]interface{}) bool {
				// Should be converted to int64 timestamp
				_, ok := params["startTime"].(int64)
				return ok
			},
		},
		{
			name: "date to timestamp with end of day",
			params: map[string]interface{}{
				"endTime": "2024-01-15",
			},
			transforms: []RequestTransformConfig{
				{
					Field:    "endTime",
					From:     "date",
					To:       "timestamp",
					Format:   "2006-01-02",
					EndOfDay: true,
				},
			},
			wantErr: false,
			check: func(params map[string]interface{}) bool {
				_, ok := params["endTime"].(int64)
				return ok
			},
		},
		{
			name: "string to array",
			params: map[string]interface{}{
				"ids": "1,2,3",
			},
			transforms: []RequestTransformConfig{
				{
					Field:  "ids",
					From:   "string",
					To:     "array",
					Format: ",",
				},
			},
			wantErr: false,
			check: func(params map[string]interface{}) bool {
				arr, ok := params["ids"].([]string)
				return ok && len(arr) == 3
			},
		},
		{
			name: "empty transform",
			params: map[string]interface{}{
				"test": "value",
			},
			transforms: []RequestTransformConfig{},
			wantErr:    false,
			check: func(params map[string]interface{}) bool {
				return params["test"] == "value"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rb.applyTransforms(tt.params, tt.transforms)
			if (err != nil) != tt.wantErr {
				t.Errorf("applyTransforms() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil && !tt.check(tt.params) {
				t.Errorf("applyTransforms() result check failed")
			}
		})
	}
}

func TestRequestBuilder_BuildPagination(t *testing.T) {
	rb, err := NewRequestBuilder("Asia/Shanghai", 10)
	if err != nil {
		t.Fatalf("NewRequestBuilder failed: %v", err)
	}

	config := PaginationConfig{
		Enabled: true,
		Request: struct {
			OffsetField string `yaml:"offsetField"`
			LimitField  string `yaml:"limitField"`
			PageField   string `yaml:"pageField,omitempty"`
		}{
			OffsetField: "offset",
			LimitField:  "limit",
		},
	}

	tests := []struct {
		name     string
		page     int
		pageSize int
		wantOff  int
		wantLim  int
	}{
		{
			name:     "first page default size",
			page:     1,
			pageSize: 10,
			wantOff:  0,
			wantLim:  10,
		},
		{
			name:     "second page",
			page:     2,
			pageSize: 10,
			wantOff:  10,
			wantLim:  10,
		},
		{
			name:     "invalid page",
			page:     0,
			pageSize: 10,
			wantOff:  0,
			wantLim:  10,
		},
		{
			name:     "invalid page size",
			page:     1,
			pageSize: 0,
			wantOff:  0,
			wantLim:  10,
		},
		{
			name:     "custom page size",
			page:     3,
			pageSize: 20,
			wantOff:  40,
			wantLim:  20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			off, lim := rb.BuildPagination(tt.page, tt.pageSize, config)
			if off != tt.wantOff {
				t.Errorf("BuildPagination() offset = %v, want %v", off, tt.wantOff)
			}
			if lim != tt.wantLim {
				t.Errorf("BuildPagination() limit = %v, want %v", lim, tt.wantLim)
			}
		})
	}
}

func TestToGeneratedParams(t *testing.T) {
	type TestParams struct {
		Name    string `json:"name"`
		Age     int    `json:"age"`
		Email   string `json:"email,omitempty"`
	}

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
		check   func(TestParams) bool
	}{
		{
			name: "all fields",
			params: map[string]interface{}{
				"name":  "John",
				"age":   30,
				"email": "john@example.com",
			},
			wantErr: false,
			check: func(p TestParams) bool {
				return p.Name == "John" && p.Age == 30 && p.Email == "john@example.com"
			},
		},
		{
			name: "missing optional field",
			params: map[string]interface{}{
				"name": "Jane",
				"age":  25,
			},
			wantErr: false,
			check: func(p TestParams) bool {
				return p.Name == "Jane" && p.Age == 25 && p.Email == ""
			},
		},
		{
			name:    "empty params",
			params:  map[string]interface{}{},
			wantErr: false,
			check: func(p TestParams) bool {
				return p.Name == "" && p.Age == 0
			},
		},
		{
			name:    "nil params",
			params:  nil,
			wantErr: false,
			check: func(p TestParams) bool {
				return p.Name == "" && p.Age == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target TestParams
			err := ToGeneratedParams(tt.params, &target)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToGeneratedParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil && !tt.check(target) {
				t.Errorf("ToGeneratedParams() check failed: %+v", target)
			}
		})
	}
}

func TestToGeneratedParams_WithPointers(t *testing.T) {
	type TestParams struct {
		Name  *string `json:"name"`
		Count *int    `json:"count"`
	}

	name := "test"
	count := 5

	params := map[string]interface{}{
		"name":  name,
		"count": count,
	}

	var target TestParams
	err := ToGeneratedParams(params, &target)
	if err != nil {
		t.Fatalf("ToGeneratedParams() error = %v", err)
	}

	if target.Name == nil || *target.Name != "test" {
		t.Errorf("Name not set correctly")
	}
	if target.Count == nil || *target.Count != 5 {
		t.Errorf("Count not set correctly")
	}
}
