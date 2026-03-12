package tableview

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type scalarStruct struct {
	Name   string  `json:"name"`
	Age    int     `json:"age"`
	Active bool    `json:"is_active"`
	Score  float64 `json:"score"`
}

type nestedStruct struct {
	ID     string `json:"id"`
	Config struct {
		Key string
	}
	Label string `json:"label"`
}

type manyFieldsStruct struct {
	F1  string `json:"f1"`
	F2  string `json:"f2"`
	F3  string `json:"f3"`
	F4  string `json:"f4"`
	F5  string `json:"f5"`
	F6  string `json:"f6"`
	F7  string `json:"f7"`
	F8  string `json:"f8"`
	F9  string `json:"f9"`
	F10 string `json:"f10"`
}

type noExportedFields struct {
	hidden string //nolint:unused
}

type jsonTagStruct struct {
	WorkspaceID string `json:"workspace_id"`
	DisplayName string `json:"display_name"`
	NoTag       string
}

func TestAutoDetectScalarFields(t *testing.T) {
	iter := &fakeIterator[scalarStruct]{items: []scalarStruct{{Name: "alice", Age: 30, Active: true, Score: 9.5}}}
	cfg := AutoDetect[scalarStruct](iter)
	require.NotNil(t, cfg)
	assert.Len(t, cfg.Columns, 4)
	assert.Equal(t, "Name", cfg.Columns[0].Header)
	assert.Equal(t, "Age", cfg.Columns[1].Header)
	assert.Equal(t, "Is Active", cfg.Columns[2].Header)
	assert.Equal(t, "Score", cfg.Columns[3].Header)

	val := scalarStruct{Name: "bob", Age: 25, Active: false, Score: 7.2}
	assert.Equal(t, "bob", cfg.Columns[0].Extract(val))
	assert.Equal(t, "25", cfg.Columns[1].Extract(val))
	assert.Equal(t, "false", cfg.Columns[2].Extract(val))
	assert.Equal(t, "7.2", cfg.Columns[3].Extract(val))
}

func TestAutoDetectSkipsNestedFields(t *testing.T) {
	iter := &fakeIterator[nestedStruct]{items: []nestedStruct{{ID: "123", Label: "test"}}}
	cfg := AutoDetect[nestedStruct](iter)
	require.NotNil(t, cfg)
	assert.Len(t, cfg.Columns, 2)
	assert.Equal(t, "ID", cfg.Columns[0].Header)
	assert.Equal(t, "Label", cfg.Columns[1].Header)
}

func TestAutoDetectPointerType(t *testing.T) {
	iter := &fakeIterator[*scalarStruct]{items: []*scalarStruct{{Name: "ptr", Age: 1}}}
	cfg := AutoDetect[*scalarStruct](iter)
	require.NotNil(t, cfg)
	assert.Len(t, cfg.Columns, 4)

	val := &scalarStruct{Name: "ptr", Age: 1}
	assert.Equal(t, "ptr", cfg.Columns[0].Extract(val))
	assert.Equal(t, "1", cfg.Columns[1].Extract(val))
}

func TestAutoDetectCappedAtMaxColumns(t *testing.T) {
	iter := &fakeIterator[manyFieldsStruct]{items: []manyFieldsStruct{{}}}
	cfg := AutoDetect[manyFieldsStruct](iter)
	require.NotNil(t, cfg)
	assert.Len(t, cfg.Columns, maxAutoColumns)
}

func TestAutoDetectNoExportedFields(t *testing.T) {
	iter := &fakeIterator[noExportedFields]{items: []noExportedFields{{}}}
	cfg := AutoDetect[noExportedFields](iter)
	assert.Nil(t, cfg)
}

func TestAutoDetectJsonTags(t *testing.T) {
	iter := &fakeIterator[jsonTagStruct]{items: []jsonTagStruct{{}}}
	cfg := AutoDetect[jsonTagStruct](iter)
	require.NotNil(t, cfg)
	assert.Equal(t, "Workspace ID", cfg.Columns[0].Header)
	assert.Equal(t, "Display Name", cfg.Columns[1].Header)
	assert.Equal(t, "NoTag", cfg.Columns[2].Header)
}

func TestAutoDetectCaching(t *testing.T) {
	iter1 := &fakeIterator[scalarStruct]{items: []scalarStruct{{}}}
	cfg1 := AutoDetect[scalarStruct](iter1)

	iter2 := &fakeIterator[scalarStruct]{items: []scalarStruct{{}}}
	cfg2 := AutoDetect[scalarStruct](iter2)

	// Should return the same cached pointer.
	assert.Same(t, cfg1, cfg2)
}

func TestSnakeToTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"workspace_id", "Workspace ID"},
		{"display_name", "Display Name"},
		{"id", "ID"},
		{"simple", "Simple"},
		{"a_b_c", "A B C"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, snakeToTitle(tt.input))
	}
}
