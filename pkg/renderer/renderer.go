// Package renderer provides unified output rendering for CLI
package renderer

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
)

// Format represents output format type
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

// Cell represents a single table cell
type Cell struct {
	Value  string
	Align  int // -1: left, 0: default, 1: right
	Color  string // ANSI color code
}

// Row represents a table row
type Row struct {
	Cells []Cell
}

// Table represents a formatted table
type Table struct {
	Headers []string
	Rows    []Row
}

// Renderer handles output formatting
type Renderer struct {
	format Format
}

// New creates a new renderer
func New(format Format) *Renderer {
	if format == "" {
		format = FormatTable
	}
	return &Renderer{format: format}
}

// SetFormat changes output format
func (r *Renderer) SetFormat(format Format) {
	r.format = format
}

// Render renders data according to configured format
func (r *Renderer) Render(data interface{}) error {
	switch r.format {
	case FormatJSON:
		return r.renderJSON(data)
	case FormatYAML:
		return r.renderYAML(data)
	default:
		return r.renderTable(data)
	}
}

// renderJSON outputs as JSON
func (r *Renderer) renderJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// renderYAML outputs as YAML (simple implementation)
func (r *Renderer) renderYAML(data interface{}) error {
	// Simple YAML-like output for now
	return r.renderJSON(data)
}

// renderTable renders data as a formatted table
func (r *Renderer) renderTable(data interface{}) error {
	switch v := data.(type) {
	case *Table:
		return r.renderTableStruct(v)
	case []map[string]interface{}:
		return r.renderMapSlice(v)
	case map[string]interface{}:
		return r.renderMap(v)
	default:
		// Try to convert struct/slice to table
		table := r.autoTable(data)
		if table != nil {
			return r.renderTableStruct(table)
		}
		// Fallback to JSON
		return r.renderJSON(data)
	}
}

// renderTableStruct renders a Table struct
func (r *Renderer) renderTableStruct(table *Table) error {
	if len(table.Rows) == 0 {
		fmt.Println("(empty)")
		return nil
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
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
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

// renderMapSlice renders a slice of maps
func (r *Renderer) renderMapSlice(data []map[string]interface{}) error {
	if len(data) == 0 {
		fmt.Println("(empty)")
		return nil
	}

	// Extract headers from first map
	var headers []string
	for k := range data[0] {
		headers = append(headers, k)
	}

	table := &Table{
		Headers: headers,
	}

	for _, m := range data {
		row := Row{Cells: make([]Cell, len(headers))}
		for i, h := range headers {
			row.Cells[i] = Cell{Value: fmt.Sprintf("%v", m[h])}
		}
		table.Rows = append(table.Rows, row)
	}

	return r.renderTableStruct(table)
}

// renderMap renders a single map as key-value pairs
func (r *Renderer) renderMap(data map[string]interface{}) error {
	var maxKeyLen int
	for k := range data {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}

	for k, v := range data {
		fmt.Printf("%s: %v\n", padRight(k, maxKeyLen), v)
	}
	return nil
}

// autoTable creates a Table from a struct or slice
func (r *Renderer) autoTable(data interface{}) *Table {
	v := reflect.ValueOf(data)

	// Dereference pointer
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice:
		return r.sliceToTable(v)
	case reflect.Struct:
		return r.structToTable(v)
	default:
		return nil
	}
}

// sliceToTable converts a slice to Table
func (r *Renderer) sliceToTable(v reflect.Value) *Table {
	if v.Len() == 0 {
		return nil
	}

	// Get headers from first element
	elem := v.Index(0)
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}

	if elem.Kind() != reflect.Struct {
		return nil
	}

	headers := r.getHeaders(elem.Type())
	table := &Table{Headers: headers}

	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}
		row := r.structToRow(elem)
		table.Rows = append(table.Rows, row)
	}

	return table
}

// structToTable converts a single struct to Table
func (r *Renderer) structToTable(v reflect.Value) *Table {
	if v.Kind() != reflect.Struct {
		return nil
	}

	headers := r.getHeaders(v.Type())
	row := r.structToRow(v)

	return &Table{
		Headers: headers,
		Rows:    []Row{row},
	}
}

// getHeaders extracts column headers from struct type
func (r *Renderer) getHeaders(t reflect.Type) []string {
	var headers []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" { // Skip unexported
			continue
		}
		name := field.Tag.Get("cli")
		if name == "" {
			name = field.Tag.Get("json")
			if name != "" {
				// Remove omitempty
				if idx := strings.Index(name, ","); idx != -1 {
					name = name[:idx]
				}
			} else {
				name = field.Name
			}
		}
		headers = append(headers, name)
	}
	return headers
}

// structToRow converts a struct to a table Row
func (r *Renderer) structToRow(v reflect.Value) Row {
	var cells []Cell
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" { // Skip unexported
			continue
		}

		fv := v.Field(i)
		value := r.formatValue(fv)
		cells = append(cells, Cell{Value: value})
	}

	return Row{Cells: cells}
}

// formatValue formats a reflect.Value as string
func (r *Renderer) formatValue(v reflect.Value) string {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "-"
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		s := v.String()
		if s == "" {
			return "-"
		}
		return s
	case reflect.Int, reflect.Int64, reflect.Int32:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Bool:
		if v.Bool() {
			return "yes"
		}
		return "no"
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// padRight pads string to fixed width
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// RenderResult is a helper to render operation results
func RenderResult(data interface{}, total int, format Format) error {
	r := New(format)

	// Add summary for list results
	if total > 0 {
		switch v := data.(type) {
		case []interface{}:
			fmt.Printf("总计: %d 条\n\n", total)
			return r.Render(v)
		default:
			// Check if it's a slice using reflection
			rv := reflect.ValueOf(data)
			if rv.Kind() == reflect.Slice && rv.Len() > 0 {
				fmt.Printf("总计: %d 条\n\n", total)
			}
		}
	}

	return r.Render(data)
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	fmt.Printf("✓ %s\n", message)
}

// PrintError prints an error message
func PrintError(err error) {
	fmt.Fprintf(os.Stderr, "✗ %s\n", err)
}

// PrintKV prints key-value pairs
func PrintKV(pairs map[string]string) {
	var maxLen int
	for k := range pairs {
		if len(k) > maxLen {
			maxLen = len(k)
		}
	}
	for k, v := range pairs {
		fmt.Printf("  %s: %s\n", padRight(k, maxLen), v)
	}
}
