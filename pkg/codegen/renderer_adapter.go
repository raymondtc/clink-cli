package codegen

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
)

// RendererAdapter adapts Table to the renderer package
type RendererAdapter struct {
	Format string
	Output io.Writer
}

// NewRendererAdapter creates a new RendererAdapter
func NewRendererAdapter(format string) *RendererAdapter {
	if format == "" {
		format = "table"
	}
	return &RendererAdapter{
		Format: format,
		Output: os.Stdout,
	}
}

// SetOutput sets the output writer
func (ra *RendererAdapter) SetOutput(w io.Writer) {
	ra.Output = w
}

// Render renders a table
func (ra *RendererAdapter) Render(table *Table) error {
	if table == nil || len(table.Rows) == 0 {
		fmt.Fprintln(ra.Output, "(empty)")
		return nil
	}

	switch ra.Format {
	case "json":
		return ra.renderJSON(table)
	case "csv":
		return ra.renderCSV(table)
	default:
		return ra.renderTable(table)
	}
}

// renderTable renders as formatted table
func (ra *RendererAdapter) renderTable(table *Table) error {
	if len(table.Headers) == 0 {
		return fmt.Errorf("no headers defined")
	}

	// Calculate column widths
	widths := make([]int, len(table.Headers))
	for i, h := range table.Headers {
		widths[i] = len(h)
	}
	for _, row := range table.Rows {
		for i, cell := range row.Cells {
			if i < len(widths) && len(cell.Value) > widths[i] {
				widths[i] = len(cell.Value)
			}
		}
	}

	// Create tabwriter
	w := tabwriter.NewWriter(ra.Output, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Write headers
	for i, h := range table.Headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprintf(w, "%s", padRight(h, widths[i]))
	}
	fmt.Fprintln(w)

	// Write separator
	for i := range table.Headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprintf(w, "%s", strings.Repeat("-", widths[i]))
	}
	fmt.Fprintln(w)

	// Write rows
	for _, row := range table.Rows {
		for i, cell := range row.Cells {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			if i < len(widths) {
				fmt.Fprintf(w, "%s", padRight(cell.Value, widths[i]))
			} else {
				fmt.Fprintf(w, "%s", cell.Value)
			}
		}
		fmt.Fprintln(w)
	}

	return nil
}

// renderJSON renders as JSON
func (ra *RendererAdapter) renderJSON(table *Table) error {
	// Convert table to array of maps
	data := make([]map[string]string, 0, len(table.Rows))
	for _, row := range table.Rows {
		item := make(map[string]string)
		for i, cell := range row.Cells {
			if i < len(table.Headers) {
				item[table.Headers[i]] = cell.Value
			}
		}
		data = append(data, item)
	}

	encoder := json.NewEncoder(ra.Output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// renderCSV renders as CSV
func (ra *RendererAdapter) renderCSV(table *Table) error {
	// Write headers
	fmt.Fprintln(ra.Output, strings.Join(table.Headers, ","))

	// Write rows
	for _, row := range table.Rows {
		values := make([]string, len(row.Cells))
		for i, cell := range row.Cells {
			// Escape values containing commas or quotes
			value := cell.Value
			if strings.Contains(value, ",") || strings.Contains(value, "\"") {
				value = fmt.Sprintf("\"%s\"", strings.ReplaceAll(value, "\"", "\"\""))
			}
			values[i] = value
		}
		fmt.Fprintln(ra.Output, strings.Join(values, ","))
	}

	return nil
}

// RenderList renders a list response with total
func (ra *RendererAdapter) RenderList(items []map[string]interface{}, total int, config OutputConfig) error {
	if total > 0 {
		fmt.Fprintf(ra.Output, "总计: %d 条\n\n", total)
	}

	if len(items) == 0 {
		fmt.Fprintln(ra.Output, "(empty)")
		return nil
	}

	// Build table
	table := &Table{
		Headers: config.Columns,
		Rows:    make([]TableRow, 0, len(items)),
	}

	// If no columns specified, use all keys from first item
	if len(table.Headers) == 0 {
		for k := range items[0] {
			table.Headers = append(table.Headers, k)
		}
	}

	// Build rows
	for _, item := range items {
		row := TableRow{
			Cells: make([]TableCell, 0, len(table.Headers)),
		}

		for _, header := range table.Headers {
			value := "-"
			if v, ok := item[header]; ok && v != nil {
				value = fmt.Sprintf("%v", v)
			}
			row.Cells = append(row.Cells, TableCell{Value: value})
		}

		table.Rows = append(table.Rows, row)
	}

	return ra.Render(table)
}

// RenderSingle renders a single item response
func (ra *RendererAdapter) RenderSingle(item map[string]interface{}, config OutputConfig) error {
	if item == nil {
		fmt.Fprintln(ra.Output, "(empty)")
		return nil
	}

	// Determine columns to display
	columns := config.Columns
	if len(columns) == 0 {
		for k := range item {
			columns = append(columns, k)
		}
	}

	// Find max key length for alignment
	maxLen := 0
	for _, col := range columns {
		if len(col) > maxLen {
			maxLen = len(col)
		}
	}

	// Print key-value pairs
	for _, col := range columns {
		value := "-"
		if v, ok := item[col]; ok && v != nil {
			value = fmt.Sprintf("%v", v)
		}
		fmt.Fprintf(ra.Output, "  %s: %s\n", padRight(col, maxLen), value)
	}

	return nil
}

// RenderSuccess renders a success message
func (ra *RendererAdapter) RenderSuccess(message string) {
	fmt.Fprintf(ra.Output, "✓ %s\n", message)
}

// RenderError renders an error message
func (ra *RendererAdapter) RenderError(err error) {
	fmt.Fprintf(ra.Output, "✗ %s\n", err)
}

// RenderResponse renders a response based on its type
func (ra *RendererAdapter) RenderResponse(resp interface{}, config ResponseConfig) error {
	if config.Type == "simple" {
		// Simple response just shows success message
		if config.SuccessMessage != "" {
			ra.RenderSuccess(config.SuccessMessage)
		}
		return nil
	}

	parser, err := NewResponseParser("Asia/Shanghai")
	if err != nil {
		return err
	}

	switch config.Type {
	case "list", "paged":
		items, total, err := parser.ParseListResponse(resp, config)
		if err != nil {
			return err
		}
		return ra.RenderList(items, total, config.Output)

	case "single":
		item, err := parser.ParseSingleResponse(resp, config)
		if err != nil {
			return err
		}
		return ra.RenderSingle(item, config.Output)

	default:
		return fmt.Errorf("unsupported response type: %s", config.Type)
	}
}

// TableBuilder helps build tables programmatically
type TableBuilder struct {
	headers []string
	rows    []TableRow
}

// NewTableBuilder creates a new TableBuilder
func NewTableBuilder() *TableBuilder {
	return &TableBuilder{
		headers: []string{},
		rows:    []TableRow{},
	}
}

// SetHeaders sets the table headers
func (tb *TableBuilder) SetHeaders(headers []string) *TableBuilder {
	tb.headers = headers
	return tb
}

// AddRow adds a row from a map
func (tb *TableBuilder) AddRow(data map[string]interface{}) *TableBuilder {
	row := TableRow{
		Cells: make([]TableCell, 0, len(tb.headers)),
	}

	for _, header := range tb.headers {
		value := "-"
		if v, ok := data[header]; ok && v != nil {
			value = fmt.Sprintf("%v", v)
		}
		row.Cells = append(row.Cells, TableCell{Value: value})
	}

	tb.rows = append(tb.rows, row)
	return tb
}

// AddStruct adds a row from a struct
func (tb *TableBuilder) AddStruct(data interface{}) *TableBuilder {
	item, err := structToMap(data)
	if err != nil {
		return tb
	}
	return tb.AddRow(item)
}

// Build builds the table
func (tb *TableBuilder) Build() *Table {
	return &Table{
		Headers: tb.headers,
		Rows:    tb.rows,
	}
}

// structToMap converts a struct to a map
func structToMap(data interface{}) (map[string]interface{}, error) {
	if data == nil {
		return nil, nil
	}

	// If it's already a map
	if m, ok := data.(map[string]interface{}); ok {
		return m, nil
	}

	v := reflect.ValueOf(data)

	// Dereference pointer
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}

	// Must be a struct
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %v", v.Kind())
	}

	result := make(map[string]interface{})
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Get field name from json tag or field name
		name := field.Tag.Get("json")
		if name == "" {
			name = field.Name
		} else {
			if idx := strings.Index(name, ","); idx != -1 {
				name = name[:idx]
			}
		}

		// Unwrap pointer
		if value.Kind() == reflect.Ptr {
			if value.IsNil() {
				continue
			}
			value = value.Elem()
		}

		result[name] = value.Interface()
	}

	return result, nil
}

// padRight pads a string to the right
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// RenderFromSlice renders data from a slice
func RenderFromSlice(data interface{}, format string) error {
	adapter := NewRendererAdapter(format)

	// Convert to table
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("expected slice, got %v", v.Kind())
	}

	if v.Len() == 0 {
		fmt.Println("(empty)")
		return nil
	}

	// Get headers from first element
	elem := v.Index(0)
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}

	item, err := structToMap(elem.Interface())
	if err != nil {
		return err
	}

	tb := NewTableBuilder()
	headers := make([]string, 0, len(item))
	for k := range item {
		headers = append(headers, k)
	}
	tb.SetHeaders(headers)

	// Add all elements
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}
		tb.AddStruct(elem.Interface())
	}

	return adapter.Render(tb.Build())
}
