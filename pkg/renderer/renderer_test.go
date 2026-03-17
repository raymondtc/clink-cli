package renderer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	r := New(FormatTable)
	assert.NotNil(t, r)
	assert.Equal(t, FormatTable, r.format)

	r2 := New(FormatJSON)
	assert.Equal(t, FormatJSON, r2.format)
}

func TestFormatDefault(t *testing.T) {
	r := New("")
	assert.Equal(t, FormatTable, r.format)
}

func TestPadRight(t *testing.T) {
	assert.Equal(t, "hello     ", padRight("hello", 10))
	assert.Equal(t, "hello", padRight("hello", 5))
	assert.Equal(t, "hello", padRight("hello", 3))
}



func TestRenderTableStruct(t *testing.T) {
	table := &Table{
		Headers: []string{"Name", "Age"},
		Rows: []Row{
			{Cells: []Cell{{Value: "Alice"}, {Value: "30"}}},
			{Cells: []Cell{{Value: "Bob"}, {Value: "25"}}},
		},
	}

	r := New(FormatTable)
	err := r.Render(table)
	assert.NoError(t, err)
}

func TestRenderEmptyTable(t *testing.T) {
	table := &Table{
		Headers: []string{"Name"},
		Rows:    []Row{},
	}

	r := New(FormatTable)
	err := r.Render(table)
	assert.NoError(t, err)
}

func TestPrintKV(t *testing.T) {
	pairs := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	// Just ensure it doesn't panic
	PrintKV(pairs)
}

func TestPrintSuccess(t *testing.T) {
	// Just ensure it doesn't panic
	PrintSuccess("Test message")
}
