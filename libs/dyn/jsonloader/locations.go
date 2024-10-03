package jsonloader

import (
	"sort"

	"github.com/databricks/cli/libs/dyn"
)

type LineOffset struct {
	Line  int
	Start int64
}

// buildLineOffsets scans the input data and records the starting byte offset of each line.
func BuildLineOffsets(data []byte) []LineOffset {
	offsets := []LineOffset{{Line: 1, Start: 0}}
	line := 1
	for i, b := range data {
		if b == '\n' {
			line++
			offsets = append(offsets, LineOffset{Line: line, Start: int64(i + 1)})
		}
	}
	return offsets
}

// GetPosition maps a byte offset to its corresponding line and column numbers.
func GetPosition(offset int64, offsets []LineOffset) dyn.Location {
	// Binary search to find the line
	idx := sort.Search(len(offsets), func(i int) bool {
		return offsets[i].Start > offset
	}) - 1

	if idx < 0 {
		idx = 0
	}

	lineOffset := offsets[idx]
	return dyn.Location{
		File:   "(inline)",
		Line:   lineOffset.Line,
		Column: int(offset-lineOffset.Start) + 1,
	}
}
