package codegen

import (
	"bytes"
	"strings"
	"testing"
)

func TestRendererAdapter_Render(t *testing.T) {
	tests := []struct {
		name   string
		format string
		table  *Table
		want   string
	}{
		{
			name:   "empty table",
			format: "table",
			table:  &Table{Headers: []string{"A"}, Rows: []TableRow{}},
			want:   "(empty)",
		},
		{
			name:   "simple table",
			format: "table",
			table: &Table{
				Headers: []string{"Name", "Value"},
				Rows: []TableRow{
					{Cells: []TableCell{{Value: "test1"}, {Value: "100"}}},
				},
			},
			want: "test1",
		},
		{
			name:   "json format",
			format: "json",
			table: &Table{
				Headers: []string{"Name"},
				Rows: []TableRow{
					{Cells: []TableCell{{Value: "test"}}},
				},
			},
			want: `"Name"`,
		},
		{
			name:   "csv format",
			format: "csv",
			table: &Table{
				Headers: []string{"Name", "Value"},
				Rows: []TableRow{
					{Cells: []TableCell{{Value: "test"}, {Value: "100"}}},
				},
			},
			want: "Name,Value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ra := NewRendererAdapter(tt.format)
			var buf bytes.Buffer
			ra.SetOutput(&buf)

			err := ra.Render(tt.table)
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}

			got := buf.String()
			if !strings.Contains(got, tt.want) {
				t.Errorf("Render() output = %v, want to contain %v", got, tt.want)
			}
		})
	}
}

func TestRendererAdapter_RenderList(t *testing.T) {
	ra := NewRendererAdapter("table")
	var buf bytes.Buffer
	ra.SetOutput(&buf)

	items := []map[string]interface{}{
		{"Name": "Item1", "Value": 100},
		{"Name": "Item2", "Value": 200},
	}

	config := OutputConfig{
		Columns: []string{"Name", "Value"},
	}

	err := ra.RenderList(items, 10, config)
	if err != nil {
		t.Errorf("RenderList() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Item1") {
		t.Error("RenderList() should contain Item1")
	}
	if !strings.Contains(output, "总计: 10 条") {
		t.Error("RenderList() should contain total count")
	}
}

func TestRendererAdapter_RenderList_Empty(t *testing.T) {
	ra := NewRendererAdapter("table")
	var buf bytes.Buffer
	ra.SetOutput(&buf)

	err := ra.RenderList([]map[string]interface{}{}, 0, OutputConfig{})
	if err != nil {
		t.Errorf("RenderList() error = %v", err)
	}

	if !strings.Contains(buf.String(), "(empty)") {
		t.Error("RenderList() should show empty message")
	}
}

func TestRendererAdapter_RenderSingle(t *testing.T) {
	ra := NewRendererAdapter("table")
	var buf bytes.Buffer
	ra.SetOutput(&buf)

	item := map[string]interface{}{
		"Name":  "Test",
		"Value": 100,
	}

	config := OutputConfig{
		Columns: []string{"Name", "Value"},
	}

	err := ra.RenderSingle(item, config)
	if err != nil {
		t.Errorf("RenderSingle() error = %v", err)
	}

	output := buf.String()
	// padRight adds spaces for alignment, so check for "Name" not "Name:"
	if !strings.Contains(output, "Name") {
		t.Errorf("RenderSingle() should contain Name, output was: %q", output)
	}
}

func TestRendererAdapter_RenderSingle_Empty(t *testing.T) {
	ra := NewRendererAdapter("table")
	var buf bytes.Buffer
	ra.SetOutput(&buf)

	err := ra.RenderSingle(nil, OutputConfig{})
	if err != nil {
		t.Errorf("RenderSingle() error = %v", err)
	}

	if !strings.Contains(buf.String(), "(empty)") {
		t.Error("RenderSingle() should show empty message")
	}
}

func TestRendererAdapter_RenderSuccess(t *testing.T) {
	ra := NewRendererAdapter("table")
	var buf bytes.Buffer
	ra.SetOutput(&buf)

	ra.RenderSuccess("Operation completed")

	if !strings.Contains(buf.String(), "✓") {
		t.Error("RenderSuccess() should contain checkmark")
	}
	if !strings.Contains(buf.String(), "Operation completed") {
		t.Error("RenderSuccess() should contain message")
	}
}

func TestRendererAdapter_RenderError(t *testing.T) {
	ra := NewRendererAdapter("table")
	var buf bytes.Buffer
	ra.SetOutput(&buf)

	ra.RenderError(&APIError{StatusCode: 500, Message: "Server error"})

	if !strings.Contains(buf.String(), "✗") {
		t.Error("RenderError() should contain X mark")
	}
	if !strings.Contains(buf.String(), "Server error") {
		t.Error("RenderError() should contain error message")
	}
}

func TestTableBuilder(t *testing.T) {
	t.Run("basic building", func(t *testing.T) {
		tb := NewTableBuilder()
		tb.SetHeaders([]string{"Col1", "Col2"})
		tb.AddRow(map[string]interface{}{
			"Col1": "value1",
			"Col2": "value2",
		})

		table := tb.Build()
		if len(table.Headers) != 2 {
			t.Errorf("Expected 2 headers, got %d", len(table.Headers))
		}
		if len(table.Rows) != 1 {
			t.Errorf("Expected 1 row, got %d", len(table.Rows))
		}
	})

	t.Run("add struct", func(t *testing.T) {
		type TestStruct struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}

		tb := NewTableBuilder()
		tb.SetHeaders([]string{"id", "name"})
		tb.AddStruct(TestStruct{ID: 1, Name: "test"})

		table := tb.Build()
		if len(table.Rows) != 1 {
			t.Errorf("Expected 1 row, got %d", len(table.Rows))
		}
	})
}

func TestStructToMap(t *testing.T) {
	type TestStruct struct {
		ID     int     `json:"id"`
		Name   string  `json:"name"`
		Hidden string  // No json tag
	}

	tests := []struct {
		name string
		data interface{}
		want map[string]interface{}
	}{
		{
			name: "struct",
			data: TestStruct{ID: 1, Name: "test", Hidden: "hidden"},
			want: map[string]interface{}{
				"id":   1,
				"name": "test",
			},
		},
		{
			name: "pointer",
			data: &TestStruct{ID: 2, Name: "test2"},
			want: map[string]interface{}{
				"id":   2,
				"name": "test2",
			},
		},
		{
			name: "nil pointer",
			data: (*TestStruct)(nil),
			want: nil,
		},
		{
			name: "map",
			data: map[string]interface{}{"key": "value"},
			want: map[string]interface{}{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := structToMap(tt.data)
			if err != nil {
				t.Errorf("structToMap() error = %v", err)
				return
			}

			if tt.want == nil {
				if got != nil {
					t.Errorf("structToMap() = %v, want nil", got)
				}
				return
			}

			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("structToMap()[%s] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{"longer than width", "hello", 3, "hello"},
		{"same as width", "hello", 5, "hello"},
		{"shorter than width", "hi", 5, "hi   "},
		{"empty string", "", 5, "     "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := padRight(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("padRight() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderFromSlice(t *testing.T) {
	type Item struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	items := []Item{
		{Name: "Item1", Value: 100},
		{Name: "Item2", Value: 200},
	}

	// RenderFromSlice outputs to stdout, so we just verify no error
	err := RenderFromSlice(items, "table")
	if err != nil {
		t.Errorf("RenderFromSlice() error = %v", err)
	}
}

func TestRenderFromSlice_Empty(t *testing.T) {
	items := []struct{}{}

	err := RenderFromSlice(items, "table")
	if err != nil {
		t.Errorf("RenderFromSlice() error = %v", err)
	}
}
