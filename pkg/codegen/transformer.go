package codegen

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TimeTransformer handles time-related transformations
type TimeTransformer struct {
	DefaultTimezone *time.Location
}

// NewTimeTransformer creates a new TimeTransformer
func NewTimeTransformer(timezone string) (*TimeTransformer, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.Local
	}
	return &TimeTransformer{DefaultTimezone: loc}, nil
}

// TransformDateToTimestamp converts a date string to Unix timestamp
func (tt *TimeTransformer) TransformDateToTimestamp(value, format string, endOfDay bool) (int64, error) {
	if value == "" {
		return 0, nil
	}

	t, err := time.ParseInLocation(format, value, tt.DefaultTimezone)
	if err != nil {
		return 0, fmt.Errorf("parse date %q with format %q: %w", value, format, err)
	}

	if endOfDay {
		t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	return t.Unix(), nil
}

// TransformDateTimeToTimestamp converts a datetime string to Unix timestamp
func (tt *TimeTransformer) TransformDateTimeToTimestamp(value, format string) (int64, error) {
	if value == "" {
		return 0, nil
	}

	t, err := time.ParseInLocation(format, value, tt.DefaultTimezone)
	if err != nil {
		return 0, fmt.Errorf("parse datetime %q with format %q: %w", value, format, err)
	}

	return t.Unix(), nil
}

// FormatTimestamp formats a Unix timestamp for display
func (tt *TimeTransformer) FormatTimestamp(timestamp int64, format string) string {
	if timestamp == 0 {
		return "-"
	}
	t := time.Unix(timestamp, 0).In(tt.DefaultTimezone)
	return t.Format(format)
}

// PaginationTransformer handles pagination calculations
type PaginationTransformer struct {
	DefaultPageSize int
}

// NewPaginationTransformer creates a new PaginationTransformer
func NewPaginationTransformer(defaultPageSize int) *PaginationTransformer {
	if defaultPageSize <= 0 {
		defaultPageSize = 10
	}
	return &PaginationTransformer{DefaultPageSize: defaultPageSize}
}

// PageToOffset converts page-based pagination to offset-based
func (pt *PaginationTransformer) PageToOffset(page, pageSize int) int {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = pt.DefaultPageSize
	}
	return (page - 1) * pageSize
}

// CalculateTotalPages calculates total pages from total items and page size
func (pt *PaginationTransformer) CalculateTotalPages(total, pageSize int) int {
	if pageSize <= 0 {
		pageSize = pt.DefaultPageSize
	}
	if total <= 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}

// EnumTransformer handles enum value transformations
type EnumTransformer struct{}

// NewEnumTransformer creates a new EnumTransformer
func NewEnumTransformer() *EnumTransformer {
	return &EnumTransformer{}
}

// Transform converts a value using the enum mapping
func (et *EnumTransformer) Transform(value interface{}, mapping map[interface{}]string) string {
	if mapping == nil {
		return fmt.Sprintf("%v", value)
	}

	// Try direct lookup with the same type
	if str, ok := mapping[value]; ok {
		return str
	}

	// Try string conversion lookup
	strKey := fmt.Sprintf("%v", value)
	for k, v := range mapping {
		if fmt.Sprintf("%v", k) == strKey {
			return v
		}
	}

	return strKey
}

// TransformMap applies enum transformation to a map field
func (et *EnumTransformer) TransformMap(data map[string]interface{}, field string, mapping map[interface{}]string) {
	if value, ok := data[field]; ok {
		data[field] = et.Transform(value, mapping)
	}
}

// DurationTransformer handles duration formatting
type DurationTransformer struct{}

// NewDurationTransformer creates a new DurationTransformer
func NewDurationTransformer() *DurationTransformer {
	return &DurationTransformer{}
}

// FormatSeconds formats seconds into human-readable duration
func (dt *DurationTransformer) FormatSeconds(seconds int, format string) string {
	if seconds <= 0 {
		return "-"
	}

	if format != "" {
		// Use custom format with {{.}} placeholder
		return strings.ReplaceAll(format, "{{.}}", strconv.Itoa(seconds))
	}

	// Default formatting
	if seconds < 60 {
		return fmt.Sprintf("%d秒", seconds)
	}
	if seconds < 3600 {
		minutes := seconds / 60
		secs := seconds % 60
		if secs == 0 {
			return fmt.Sprintf("%d分钟", minutes)
		}
		return fmt.Sprintf("%d分%d秒", minutes, secs)
	}
	hours := seconds / 3600
	remaining := seconds % 3600
	minutes := remaining / 60
	secs := remaining % 60
	if minutes == 0 && secs == 0 {
		return fmt.Sprintf("%d小时", hours)
	}
	if secs == 0 {
		return fmt.Sprintf("%d小时%d分", hours, minutes)
	}
	return fmt.Sprintf("%d小时%d分%d秒", hours, minutes, secs)
}

// FieldTransformer handles general field transformations
type FieldTransformer struct {
	TimeTransformer      *TimeTransformer
	EnumTransformer      *EnumTransformer
	DurationTransformer  *DurationTransformer
}

// NewFieldTransformer creates a new FieldTransformer
func NewFieldTransformer(timezone string) (*FieldTransformer, error) {
	tt, err := NewTimeTransformer(timezone)
	if err != nil {
		return nil, err
	}
	return &FieldTransformer{
		TimeTransformer:     tt,
		EnumTransformer:     NewEnumTransformer(),
		DurationTransformer: NewDurationTransformer(),
	}, nil
}

// TransformValue transforms a single value based on its configuration
func (ft *FieldTransformer) TransformValue(value interface{}, config ResponseMappingConfig) (string, error) {
	if value == nil {
		return "-", nil
	}

	switch config.Type {
	case "datetime":
		ts, err := toInt64(value)
		if err != nil {
			return "-", fmt.Errorf("convert to timestamp: %w", err)
		}
		return ft.TimeTransformer.FormatTimestamp(ts, config.Format), nil

	case "duration":
		secs, err := toInt(value)
		if err != nil {
			return "-", fmt.Errorf("convert to seconds: %w", err)
		}
		return ft.DurationTransformer.FormatSeconds(secs, config.Format), nil

	case "enum":
		return ft.EnumTransformer.Transform(value, config.Enum), nil

	case "int":
		n, err := toInt(value)
		if err != nil {
			return "-", err
		}
		return strconv.Itoa(n), nil

	case "string":
		return fmt.Sprintf("%v", value), nil

	default:
		return fmt.Sprintf("%v", value), nil
	}
}

// Helper functions
func toInt64(v interface{}) (int64, error) {
	switch val := v.(type) {
	case int64:
		return val, nil
	case int:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case float64:
		return int64(val), nil
	case string:
		return strconv.ParseInt(val, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", v)
	}
}

func toInt(v interface{}) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case int64:
		return int(val), nil
	case int32:
		return int(val), nil
	case float64:
		return int(val), nil
	case string:
		n, err := strconv.Atoi(val)
		if err != nil {
			return 0, err
		}
		return n, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}

// NormalizePhone normalizes a phone number
func NormalizePhone(phone string) string {
	// Remove all non-digit characters except +
	var result strings.Builder
	for _, r := range phone {
		if (r >= '0' && r <= '9') || r == '+' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// SplitArray splits a comma-separated string into array
func SplitArray(value, separator string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, separator)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
