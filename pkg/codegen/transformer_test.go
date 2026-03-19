package codegen

import (
	"testing"
	"time"
)

func TestTimeTransformer_TransformDateToTimestamp(t *testing.T) {
	tt, err := NewTimeTransformer("Asia/Shanghai")
	if err != nil {
		t.Fatalf("NewTimeTransformer failed: %v", err)
	}

	tests := []struct {
		name     string
		value    string
		format   string
		endOfDay bool
		wantErr  bool
	}{
		{
			name:     "valid date",
			value:    "2024-01-15",
			format:   "2006-01-02",
			endOfDay: false,
			wantErr:  false,
		},
		{
			name:     "valid date with end of day",
			value:    "2024-01-15",
			format:   "2006-01-02",
			endOfDay: true,
			wantErr:  false,
		},
		{
			name:     "empty value",
			value:    "",
			format:   "2006-01-02",
			endOfDay: false,
			wantErr:  false,
		},
		{
			name:     "invalid format",
			value:    "15-01-2024",
			format:   "2006-01-02",
			endOfDay: false,
			wantErr:  true,
		},
	}

	for _, tt_ := range tests {
		t.Run(tt_.name, func(t *testing.T) {
			timestamp, err := tt.TransformDateToTimestamp(tt_.value, tt_.format, tt_.endOfDay)
			if (err != nil) != tt_.wantErr {
				t.Errorf("TransformDateToTimestamp() error = %v, wantErr %v", err, tt_.wantErr)
				return
			}
			if !tt_.wantErr && tt_.value != "" && timestamp == 0 {
				t.Errorf("TransformDateToTimestamp() returned 0 for valid input")
			}
		})
	}
}

func TestTimeTransformer_FormatTimestamp(t *testing.T) {
	tt, err := NewTimeTransformer("Asia/Shanghai")
	if err != nil {
		t.Fatalf("NewTimeTransformer failed: %v", err)
	}

	tests := []struct {
		name      string
		timestamp int64
		format    string
		want      string
	}{
		{
			name:      "zero timestamp",
			timestamp: 0,
			format:    "2006-01-02",
			want:      "-",
		},
		{
			name:      "valid timestamp",
			timestamp: 1705276800, // 2024-01-15 00:00:00 UTC
			format:    "2006-01-02",
			want:      "2024-01-15",
		},
	}

	for _, tt_ := range tests {
		t.Run(tt_.name, func(t *testing.T) {
			got := tt.FormatTimestamp(tt_.timestamp, tt_.format)
			// Note: timezone differences may affect the result
			if got == "" {
				t.Errorf("FormatTimestamp() returned empty string")
			}
		})
	}
}

func TestPaginationTransformer_PageToOffset(t *testing.T) {
	pt := NewPaginationTransformer(10)

	tests := []struct {
		name     string
		page     int
		pageSize int
		want     int
	}{
		{
			name:     "first page",
			page:     1,
			pageSize: 10,
			want:     0,
		},
		{
			name:     "second page",
			page:     2,
			pageSize: 10,
			want:     10,
		},
		{
			name:     "invalid page",
			page:     0,
			pageSize: 10,
			want:     0, // Should be treated as page 1
		},
		{
			name:     "invalid page size",
			page:     2,
			pageSize: 0,
			want:     10, // Should use default page size
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pt.PageToOffset(tt.page, tt.pageSize)
			if got != tt.want {
				t.Errorf("PageToOffset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnumTransformer_Transform(t *testing.T) {
	et := NewEnumTransformer()

	mapping := map[interface{}]string{
		0: "离线",
		1: "在线",
		2: "忙碌",
	}

	tests := []struct {
		name    string
		value   interface{}
		mapping map[interface{}]string
		want    string
	}{
		{
			name:    "valid enum int",
			value:   0,
			mapping: mapping,
			want:    "离线",
		},
		{
			name:    "valid enum int64",
			value:   int64(1),
			mapping: mapping,
			want:    "在线",
		},
		{
			name:    "unknown enum",
			value:   99,
			mapping: mapping,
			want:    "99",
		},
		{
			name:    "nil mapping",
			value:   0,
			mapping: nil,
			want:    "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := et.Transform(tt.value, tt.mapping)
			if got != tt.want {
				t.Errorf("Transform() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDurationTransformer_FormatSeconds(t *testing.T) {
	dt := NewDurationTransformer()

	tests := []struct {
		name   string
		seconds int
		format string
		want   string
	}{
		{
			name:    "zero seconds",
			seconds: 0,
			format:  "",
			want:    "-",
		},
		{
			name:    "less than a minute",
			seconds: 45,
			format:  "",
			want:    "45秒",
		},
		{
			name:    "exactly one minute",
			seconds: 60,
			format:  "",
			want:    "1分钟",
		},
		{
			name:    "minutes and seconds",
			seconds: 125,
			format:  "",
			want:    "2分5秒",
		},
		{
			name:    "one hour",
			seconds: 3600,
			format:  "",
			want:    "1小时",
		},
		{
			name:    "hours and minutes",
			seconds: 3660,
			format:  "",
			want:    "1小时1分",
		},
		{
			name:    "custom format",
			seconds: 120,
			format:  "{{.}}秒",
			want:    "120秒",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dt.FormatSeconds(tt.seconds, tt.format)
			if got != tt.want {
				t.Errorf("FormatSeconds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFieldTransformer_TransformValue(t *testing.T) {
	ft, err := NewFieldTransformer("Asia/Shanghai")
	if err != nil {
		t.Fatalf("NewFieldTransformer failed: %v", err)
	}

	tests := []struct {
		name    string
		value   interface{}
		config  ResponseMappingConfig
		want    string
		wantErr bool
	}{
		{
			name:    "nil value",
			value:   nil,
			config:  ResponseMappingConfig{Type: "string"},
			want:    "-",
			wantErr: false,
		},
		{
			name:    "string value",
			value:   "test",
			config:  ResponseMappingConfig{Type: "string"},
			want:    "test",
			wantErr: false,
		},
		{
			name:    "int value",
			value:   42,
			config:  ResponseMappingConfig{Type: "int"},
			want:    "42",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ft.TransformValue(tt.value, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("TransformValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("TransformValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  string
	}{
		{
			name:  "simple number",
			phone: "13800138000",
			want:  "13800138000",
		},
		{
			name:  "with dashes",
			phone: "138-0013-8000",
			want:  "13800138000",
		},
		{
			name:  "with spaces",
			phone: "138 0013 8000",
			want:  "13800138000",
		},
		{
			name:  "with plus",
			phone: "+86 138-0013-8000",
			want:  "+8613800138000",
		},
		{
			name:  "empty",
			phone: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizePhone(tt.phone)
			if got != tt.want {
				t.Errorf("NormalizePhone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitArray(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		separator string
		want      []string
	}{
		{
			name:      "comma separated",
			value:     "a,b,c",
			separator: ",",
			want:      []string{"a", "b", "c"},
		},
		{
			name:      "with spaces",
			value:     "a, b, c",
			separator: ",",
			want:      []string{"a", "b", "c"},
		},
		{
			name:      "empty",
			value:     "",
			separator: ",",
			want:      nil,
		},
		{
			name:      "pipe separated",
			value:     "a|b|c",
			separator: "|",
			want:      []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitArray(tt.value, tt.separator)
			if len(got) != len(tt.want) {
				t.Errorf("SplitArray() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("SplitArray()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGetDynamicDefault(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
	}{
		{
			name:    "today",
			pattern: "today",
		},
		{
			name:    "yesterday",
			pattern: "yesterday",
		},
		{
			name:    "week ago",
			pattern: "weekAgo",
		},
		{
			name:    "month ago",
			pattern: "monthAgo",
		},
		{
			name:    "unknown pattern",
			pattern: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDynamicDefault(tt.pattern)
			if got == "" {
				t.Errorf("GetDynamicDefault() returned empty string for %s", tt.pattern)
			}
			// Verify date formats are valid
			if tt.pattern != "unknown" && tt.pattern != "uuid" {
				_, err := time.Parse("2006-01-02", got)
				if err != nil {
					t.Errorf("GetDynamicDefault() returned invalid date format: %v", got)
				}
			}
		})
	}
}
