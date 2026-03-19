package codegen

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// RequestBuilder builds request parameters from CLI flags
type RequestBuilder struct {
	TimeTransformer       *TimeTransformer
	PaginationTransformer *PaginationTransformer
	Timezone              *time.Location
}

// NewRequestBuilder creates a new RequestBuilder
func NewRequestBuilder(timezone string, defaultPageSize int) (*RequestBuilder, error) {
	tt, err := NewTimeTransformer(timezone)
	if err != nil {
		return nil, err
	}
	return &RequestBuilder{
		TimeTransformer:       tt,
		PaginationTransformer: NewPaginationTransformer(defaultPageSize),
		Timezone:              tt.DefaultTimezone,
	}, nil
}

// BuildParams builds request parameters from flag values and config
func (rb *RequestBuilder) BuildParams(
	ctx context.Context,
	flags map[string]interface{},
	config EndpointConfig,
) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	// Process parameter fields
	for _, field := range config.Parameters.Fields {
		value, err := rb.getFlagValue(flags, field)
		if err != nil {
			return nil, fmt.Errorf("get value for %s: %w", field.Name, err)
		}
		params[field.Name] = value
	}

	// Apply transformations
	if err := rb.applyTransforms(params, config.Request.Transforms); err != nil {
		return nil, fmt.Errorf("apply transforms: %w", err)
	}

	return params, nil
}

// getFlagValue gets and validates the value for a parameter
func (rb *RequestBuilder) getFlagValue(flags map[string]interface{}, field ParameterConfig) (interface{}, error) {
	// Get value from flags (using flag name or parameter name)
	key := field.Flag
	if key == "" {
		key = field.Name
	}

	value, exists := flags[key]
	if !exists || value == nil {
		// Check for required field
		if field.Required {
			return nil, fmt.Errorf("required parameter %s is missing", field.Name)
		}
		// Use default value
		return field.Default, nil
	}

	// Validate the value
	if err := rb.validateValue(value, field); err != nil {
		return nil, err
	}

	return value, nil
}

// validateValue validates a parameter value
func (rb *RequestBuilder) validateValue(value interface{}, field ParameterConfig) error {
	if value == nil {
		if field.Required {
			return fmt.Errorf("field %s is required", field.Name)
		}
		return nil
	}

	// Check for empty string on required fields
	if field.Required {
		if str, ok := value.(string); ok && str == "" {
			return fmt.Errorf("field %s is required", field.Name)
		}
	}

	switch field.Type {
	case "string":
		// Allow any type to be converted to string
		str, ok := value.(string)
		if !ok {
			// Try to convert to string
			str = fmt.Sprintf("%v", value)
		}
		if field.Validate == "phone" && str != "" {
			phone := NormalizePhone(str)
			if len(phone) < 7 {
				return fmt.Errorf("field %s: invalid phone number", field.Name)
			}
		}

	case "int":
		switch v := value.(type) {
		case int:
			// OK
		case int64:
			// OK
		case float64:
			// OK
		case string:
			if _, err := strconv.Atoi(v); err != nil {
				return fmt.Errorf("field %s: cannot parse %q as int", field.Name, v)
			}
		default:
			return fmt.Errorf("field %s expects int, got %T", field.Name, value)
		}

	case "bool":
		switch v := value.(type) {
		case bool:
			// OK
		case string:
			if _, err := strconv.ParseBool(v); err != nil {
				return fmt.Errorf("field %s: cannot parse %q as bool", field.Name, v)
			}
		default:
			return fmt.Errorf("field %s expects bool, got %T", field.Name, value)
		}

	case "date":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("field %s expects date string, got %T", field.Name, value)
		}
		if str != "" {
			format := "2006-01-02"
			if _, err := time.Parse(format, str); err != nil {
				return fmt.Errorf("field %s: invalid date format, expected YYYY-MM-DD", field.Name)
			}
		}
	}

	return nil
}

// applyTransforms applies request transformations
func (rb *RequestBuilder) applyTransforms(params map[string]interface{}, transforms []RequestTransformConfig) error {
	for _, t := range transforms {
		value, exists := params[t.Field]
		if !exists || value == nil {
			continue
		}

		transformed, err := rb.transformValue(value, t)
		if err != nil {
			return fmt.Errorf("transform field %s: %w", t.Field, err)
		}
		params[t.Field] = transformed
	}
	return nil
}

// transformValue transforms a single value
func (rb *RequestBuilder) transformValue(value interface{}, config RequestTransformConfig) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	// Handle date to timestamp transformation
	if config.From == "date" && config.To == "timestamp" {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string for date, got %T", value)
		}
		if str == "" {
			return nil, nil
		}

		timestamp, err := rb.TimeTransformer.TransformDateToTimestamp(str, config.Format, config.EndOfDay)
		if err != nil {
			return nil, err
		}
		return timestamp, nil
	}

	// Handle datetime to timestamp transformation
	if config.From == "datetime" && config.To == "timestamp" {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string for datetime, got %T", value)
		}
		if str == "" {
			return nil, nil
		}

		timestamp, err := rb.TimeTransformer.TransformDateTimeToTimestamp(str, config.Format)
		if err != nil {
			return nil, err
		}
		return timestamp, nil
	}

	// Handle string split to array
	if config.From == "string" && config.To == "array" {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		separator := ","
		if config.Format != "" {
			separator = config.Format
		}
		return SplitArray(str, separator), nil
	}

	// Default: no transformation
	return value, nil
}

// BuildPagination builds pagination parameters
func (rb *RequestBuilder) BuildPagination(page, pageSize int, config PaginationConfig) (offset, limit int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = rb.PaginationTransformer.DefaultPageSize
	}

	offset = rb.PaginationTransformer.PageToOffset(page, pageSize)
	limit = pageSize

	return offset, limit
}

// ApplyTypeDefaults applies default values from type definitions
func (rb *RequestBuilder) ApplyTypeDefaults(
	flags map[string]interface{},
	typeName string,
	typeDef TypeDefinition,
) error {
	for fieldName, fieldConfig := range typeDef {
		flagName := fieldName
		if fieldConfig.Flag != "" {
			flagName = fieldConfig.Flag
		}

		// Only apply default if flag is not set
		if _, exists := flags[flagName]; !exists {
			if fieldConfig.Default != nil {
				flags[flagName] = fieldConfig.Default
			}
		}
	}
	return nil
}

// ToGeneratedParams converts generic params to generated type
func ToGeneratedParams(params map[string]interface{}, target interface{}) error {
	if params == nil {
		return nil
	}

	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer")
	}

	v = v.Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Get json tag name
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = fieldType.Name
		} else {
			// Handle "name,omitempty" format
			if idx := strings.Index(jsonTag, ","); idx != -1 {
				jsonTag = jsonTag[:idx]
			}
		}

		// Get value from params
		value, exists := params[jsonTag]
		if !exists {
			// Try with field name
			value, exists = params[fieldType.Name]
			if !exists {
				continue
			}
		}

		// Set value (if field is settable)
		if field.CanSet() && value != nil {
			if err := setFieldValue(field, value); err != nil {
				return fmt.Errorf("set field %s: %w", fieldType.Name, err)
			}
		}
	}

	return nil
}

// setFieldValue sets a reflect.Value from an interface{}
func setFieldValue(field reflect.Value, value interface{}) error {
	if value == nil {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(fmt.Sprintf("%v", value))

	case reflect.Int, reflect.Int64, reflect.Int32:
		n, err := toInt64(value)
		if err != nil {
			return err
		}
		field.SetInt(n)

	case reflect.Bool:
		b, err := strconv.ParseBool(fmt.Sprintf("%v", value))
		if err != nil {
			return err
		}
		field.SetBool(b)

	case reflect.Ptr:
		// Create new instance and set value
		elemType := field.Type().Elem()
		elem := reflect.New(elemType)
		if err := setFieldValue(elem.Elem(), value); err != nil {
			return err
		}
		field.Set(elem)

	case reflect.Slice:
		// Handle slice types
		if arr, ok := value.([]string); ok {
			slice := reflect.MakeSlice(field.Type(), len(arr), len(arr))
			for i, s := range arr {
				slice.Index(i).SetString(s)
			}
			field.Set(slice)
		}

	default:
		return fmt.Errorf("unsupported field type: %v", field.Kind())
	}

	return nil
}

// GetDynamicDefault generates dynamic default values
func GetDynamicDefault(pattern string) string {
	switch pattern {
	case "today":
		return time.Now().Format("2006-01-02")
	case "yesterday":
		return time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	case "weekAgo":
		return time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	case "monthAgo":
		return time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	case "uuid":
		// Simple UUID generation
		return fmt.Sprintf("%d-%d", time.Now().Unix(), time.Now().Nanosecond())
	default:
		return pattern
	}
}
