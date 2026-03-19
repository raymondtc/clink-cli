package codegen

import (
	"testing"
)

// Mock response structs for testing
type MockListResponse struct {
	Total int                    `json:"total"`
	List  []MockItem             `json:"list"`
}

type MockItem struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Status    int     `json:"status"`
	CreatedAt int64   `json:"createdAt"`
}

func TestResponseParser_ParseListResponse(t *testing.T) {
	rp, err := NewResponseParser("Asia/Shanghai")
	if err != nil {
		t.Fatalf("NewResponseParser failed: %v", err)
	}

	resp := &MockListResponse{
		Total: 2,
		List: []MockItem{
			{ID: 1, Name: "Item1", Status: 1, CreatedAt: 1705276800},
			{ID: 2, Name: "Item2", Status: 0, CreatedAt: 1705363200},
		},
	}

	config := ResponseConfig{
		Type:    "list",
		Extract: "list",
		Pagination: PaginationConfig{
			Enabled: true,
			Response: struct {
				TotalPath   string `yaml:"totalPath"`
				ItemsPath   string `yaml:"itemsPath"`
				HasMorePath string `yaml:"hasMorePath,omitempty"`
			}{
				TotalPath: "total",
				ItemsPath: "list",
			},
		},
	}

	items, total, err := rp.ParseListResponse(resp, config)
	if err != nil {
		t.Fatalf("ParseListResponse() error = %v", err)
	}

	if total != 2 {
		t.Errorf("ParseListResponse() total = %v, want %v", total, 2)
	}

	if len(items) != 2 {
		t.Errorf("ParseListResponse() items count = %v, want %v", len(items), 2)
	}

	// Check first item
	if items[0]["id"] != int64(1) {
		t.Errorf("ParseListResponse() first item id = %v, want %v", items[0]["id"], 1)
	}
}

func TestResponseParser_ParseListResponse_Empty(t *testing.T) {
	rp, err := NewResponseParser("Asia/Shanghai")
	if err != nil {
		t.Fatalf("NewResponseParser failed: %v", err)
	}

	config := ResponseConfig{
		Type:    "list",
		Extract: "list",
		Pagination: PaginationConfig{
			Enabled: true,
			Response: struct {
				TotalPath   string `yaml:"totalPath"`
				ItemsPath   string `yaml:"itemsPath"`
				HasMorePath string `yaml:"hasMorePath,omitempty"`
			}{
				TotalPath: "total",
				ItemsPath: "list",
			},
		},
	}

	// Test nil response
	items, total, err := rp.ParseListResponse(nil, config)
	if err != nil {
		t.Fatalf("ParseListResponse() error = %v", err)
	}

	if total != 0 {
		t.Errorf("ParseListResponse() total = %v, want %v", total, 0)
	}

	if len(items) != 0 {
		t.Errorf("ParseListResponse() items count = %v, want %v", len(items), 0)
	}
}

func TestResponseParser_ParseSingleResponse(t *testing.T) {
	rp, err := NewResponseParser("Asia/Shanghai")
	if err != nil {
		t.Fatalf("NewResponseParser failed: %v", err)
	}

	item := MockItem{ID: 1, Name: "Item1", Status: 1}

	config := ResponseConfig{
		Type:    "single",
		Extract: "", // Empty extract means use the whole object
	}

	result, err := rp.ParseSingleResponse(&item, config)
	if err != nil {
		t.Fatalf("ParseSingleResponse() error = %v", err)
	}

	if result["id"] != int64(1) {
		t.Errorf("ParseSingleResponse() id = %v, want %v", result["id"], 1)
	}

	if result["name"] != "Item1" {
		t.Errorf("ParseSingleResponse() name = %v, want %v", result["name"], "Item1")
	}
}

func TestResponseParser_extractData(t *testing.T) {
	rp, err := NewResponseParser("Asia/Shanghai")
	if err != nil {
		t.Fatalf("NewResponseParser failed: %v", err)
	}

	tests := []struct {
		name    string
		resp    interface{}
		path    string
		wantErr bool
	}{
		{
			name:    "root path",
			resp:    &MockListResponse{Total: 10},
			path:    "",
			wantErr: false,
		},
		{
			name:    "single field",
			resp:    &MockListResponse{Total: 10},
			path:    "total",
			wantErr: false,
		},
		{
			name:    "nested field",
			resp:    &MockListResponse{List: []MockItem{{ID: 1}}},
			path:    "list",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := rp.extractData(tt.resp, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResponseParser_toMap(t *testing.T) {
	rp, err := NewResponseParser("Asia/Shanghai")
	if err != nil {
		t.Fatalf("NewResponseParser failed: %v", err)
	}

	tests := []struct {
		name string
		data interface{}
		want map[string]interface{}
	}{
		{
			name: "struct",
			data: MockItem{ID: 1, Name: "Test"},
			want: map[string]interface{}{
				"id":        int64(1),
				"name":      "Test",
				"status":    0,
				"createdAt": int64(0),
			},
		},
		{
			name: "pointer to struct",
			data: &MockItem{ID: 2, Name: "Test2"},
			want: map[string]interface{}{
				"id":        int64(2),
				"name":      "Test2",
				"status":    0,
				"createdAt": int64(0),
			},
		},
		{
			name: "map",
			data: map[string]interface{}{"key": "value"},
			want: map[string]interface{}{"key": "value"},
		},
		{
			name: "nil",
			data: nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rp.toMap(tt.data)
			if err != nil {
				t.Errorf("toMap() error = %v", err)
				return
			}

			if tt.want == nil {
				if got != nil {
					t.Errorf("toMap() = %v, want nil", got)
				}
				return
			}

			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("toMap()[%s] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestResponseParser_applyMappings(t *testing.T) {
	rp, err := NewResponseParser("Asia/Shanghai")
	if err != nil {
		t.Fatalf("NewResponseParser failed: %v", err)
	}

	items := []map[string]interface{}{
		{"id": 1, "status": 0},
		{"id": 2, "status": 1},
	}

	mappings := []ResponseMappingConfig{
		{
			From: "id",
			To:   "编号",
			Type: "int",
		},
		{
			From: "status",
			To:   "状态",
			Type: "enum",
			Enum: map[interface{}]string{
				0: "离线",
				1: "在线",
			},
		},
	}

	result := rp.applyMappings(items, mappings)

	if len(result) != 2 {
		t.Errorf("applyMappings() returned %d items, want %d", len(result), 2)
	}

	// Check first item mapping
	if result[0]["编号"] != "1" {
		t.Errorf("applyMappings() id mapping = %v, want %v", result[0]["编号"], "1")
	}

	if result[0]["状态"] != "离线" {
		t.Errorf("applyMappings() status mapping = %v, want %v", result[0]["状态"], "离线")
	}
}

func TestToIntSafe(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  int
	}{
		{"int", 42, 42},
		{"int64", int64(42), 42},
		{"int32", int32(42), 42},
		{"float64", float64(42.5), 42},
		{"string int", "42", 42},
		{"string invalid", "invalid", 0},
		{"nil", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toIntSafe(tt.value)
			if got != tt.want {
				t.Errorf("toIntSafe() = %v, want %v", got, tt.want)
			}
		})
	}
}
