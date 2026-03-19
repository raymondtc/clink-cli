package codegen

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ResponseParser parses API responses
type ResponseParser struct {
	FieldTransformer *FieldTransformer
}

// NewResponseParser creates a new ResponseParser
func NewResponseParser(timezone string) (*ResponseParser, error) {
	ft, err := NewFieldTransformer(timezone)
	if err != nil {
		return nil, err
	}
	return &ResponseParser{FieldTransformer: ft}, nil
}

// ParseListResponse parses a list response
func (rp *ResponseParser) ParseListResponse(
	resp interface{},
	config ResponseConfig,
) ([]map[string]interface{}, int, error) {
	if resp == nil {
		return []map[string]interface{}{}, 0, nil
	}

	// Extract data from response using reflection
	data, err := rp.extractData(resp, config.Pagination.Response.ItemsPath)
	if err != nil {
		return nil, 0, fmt.Errorf("extract items: %w", err)
	}

	// Extract total
	total := 0
	if totalValue, err := rp.extractValue(resp, config.Pagination.Response.TotalPath); err == nil {
		total = toIntSafe(totalValue)
	}

	// Convert to map slice
	items, err := rp.toMapSlice(data)
	if err != nil {
		return nil, 0, fmt.Errorf("convert to map slice: %w", err)
	}

	// Apply field mappings
	if len(config.Mapping) > 0 {
		items = rp.applyMappings(items, config.Mapping)
	}

	return items, total, nil
}

// ParseSingleResponse parses a single item response
func (rp *ResponseParser) ParseSingleResponse(
	resp interface{},
	config ResponseConfig,
) (map[string]interface{}, error) {
	if resp == nil {
		return nil, nil
	}

	// Extract data
	data, err := rp.extractData(resp, config.Extract)
	if err != nil {
		return nil, fmt.Errorf("extract data: %w", err)
	}

	// Convert to map
	item, err := rp.toMap(data)
	if err != nil {
		return nil, fmt.Errorf("convert to map: %w", err)
	}

	// Apply field mappings
	if len(config.Mapping) > 0 {
		items := []map[string]interface{}{item}
		items = rp.applyMappings(items, config.Mapping)
		item = items[0]
	}

	return item, nil
}

// ParseSimpleResponse parses a simple response (success/error only)
func (rp *ResponseParser) ParseSimpleResponse(resp interface{}) error {
	if resp == nil {
		return nil
	}

	// Check for error in response
	if errValue, err := rp.extractValue(resp, "error"); err == nil && errValue != nil {
		return fmt.Errorf("%v", errValue)
	}

	return nil
}

// extractData extracts data from response using path
func (rp *ResponseParser) extractData(resp interface{}, path string) (interface{}, error) {
	if path == "" || path == "." {
		return resp, nil
	}

	parts := strings.Split(path, ".")
	current := resp

	for _, part := range parts {
		if part == "" {
			continue
		}

		value, err := rp.extractField(current, part)
		if err != nil {
			return nil, fmt.Errorf("extract field %s: %w", part, err)
		}
		if value == nil {
			return nil, nil
		}
		current = value
	}

	return current, nil
}

// extractValue extracts a value from response using path
func (rp *ResponseParser) extractValue(resp interface{}, path string) (interface{}, error) {
	return rp.extractData(resp, path)
}

// extractField extracts a single field from a struct/map
func (rp *ResponseParser) extractField(data interface{}, field string) (interface{}, error) {
	if data == nil {
		return nil, nil
	}

	v := reflect.ValueOf(data)

	// Dereference pointer
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		return rp.extractStructField(v, field)
	case reflect.Map:
		return rp.extractMapField(v, field)
	case reflect.Slice:
		// Try to extract from first element if path continues
		if v.Len() > 0 {
			return rp.extractField(v.Index(0).Interface(), field)
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("cannot extract field from %v", v.Kind())
	}
}

// extractStructField extracts a field from a struct
func (rp *ResponseParser) extractStructField(v reflect.Value, field string) (interface{}, error) {
	t := v.Type()

	// Try direct field name
	for i := 0; i < v.NumField(); i++ {
		fieldType := t.Field(i)
		fieldValue := v.Field(i)

		// Check json tag
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag != "" {
			if idx := strings.Index(jsonTag, ","); idx != -1 {
				jsonTag = jsonTag[:idx]
			}
			if strings.EqualFold(jsonTag, field) {
				return rp.unwrapValue(fieldValue), nil
			}
		}

		// Check field name
		if strings.EqualFold(fieldType.Name, field) {
			return rp.unwrapValue(fieldValue), nil
		}
	}

	return nil, nil
}

// extractMapField extracts a field from a map
func (rp *ResponseParser) extractMapField(v reflect.Value, field string) (interface{}, error) {
	key := reflect.ValueOf(field)
	value := v.MapIndex(key)
	if !value.IsValid() {
		// Try case-insensitive lookup
		for _, k := range v.MapKeys() {
			if strings.EqualFold(k.String(), field) {
				return rp.unwrapValue(v.MapIndex(k)), nil
			}
		}
		return nil, nil
	}
	return rp.unwrapValue(value), nil
}

// unwrapValue unwraps a reflect.Value
func (rp *ResponseParser) unwrapValue(v reflect.Value) interface{} {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	return v.Interface()
}

// toMapSlice converts data to a slice of maps
func (rp *ResponseParser) toMapSlice(data interface{}) ([]map[string]interface{}, error) {
	if data == nil {
		return []map[string]interface{}{}, nil
	}

	v := reflect.ValueOf(data)

	// Dereference pointer
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return []map[string]interface{}{}, nil
		}
		v = v.Elem()
	}

	// If it's already a slice
	if v.Kind() == reflect.Slice {
		result := make([]map[string]interface{}, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			item, err := rp.toMap(v.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		return result, nil
	}

	// Single item, wrap in slice
	item, err := rp.toMap(data)
	if err != nil {
		return nil, err
	}
	return []map[string]interface{}{item}, nil
}

// toMap converts a struct to a map
func (rp *ResponseParser) toMap(data interface{}) (map[string]interface{}, error) {
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
		return map[string]interface{}{"value": data}, nil
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

		result[name] = rp.unwrapValue(value)
	}

	return result, nil
}

// applyMappings applies field mappings to items
func (rp *ResponseParser) applyMappings(
	items []map[string]interface{},
	mappings []ResponseMappingConfig,
) []map[string]interface{} {
	if len(mappings) == 0 {
		return items
	}

	result := make([]map[string]interface{}, len(items))
	for i, item := range items {
		mapped := make(map[string]interface{})

		for _, mapping := range mappings {
			if value, ok := item[mapping.From]; ok {
				transformed, err := rp.FieldTransformer.TransformValue(value, mapping)
				if err != nil {
					// Keep original value on error
					mapped[mapping.To] = value
				} else {
					mapped[mapping.To] = transformed
				}
			}
		}

		result[i] = mapped
	}

	return result
}

// toIntSafe converts a value to int safely
func toIntSafe(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case int32:
		return int(val)
	case float64:
		return int(val)
	case string:
		n, _ := strconv.Atoi(val)
		return n
	default:
		return 0
	}
}

// ExtractAndRender extracts data and renders it
func (rp *ResponseParser) ExtractAndRender(
	resp interface{},
	config ResponseConfig,
) (*Table, error) {
	var items []map[string]interface{}
	var err error

	switch config.Type {
	case "list", "paged":
		items, _, err = rp.ParseListResponse(resp, config)
		if err != nil {
			return nil, err
		}
	case "single":
		item, err := rp.ParseSingleResponse(resp, config)
		if err != nil {
			return nil, err
		}
		if item != nil {
			items = []map[string]interface{}{item}
		}
	default:
		return nil, fmt.Errorf("unsupported response type: %s", config.Type)
	}

	// Build table
	table := &Table{
		Headers: config.Output.Columns,
		Rows:    make([]TableRow, 0, len(items)),
	}

	// If no columns specified, use all keys from first item
	if len(table.Headers) == 0 && len(items) > 0 {
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

	return table, nil
}
