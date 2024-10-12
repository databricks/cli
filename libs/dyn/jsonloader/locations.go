package jsonloader

import (
	"sort"

	"github.com/databricks/cli/libs/dyn"
)

type LineOffset struct {
	Line  int
	Start int64
}

type Offset struct {
	offsets []LineOffset
	source  string
}

// buildLineOffsets scans the input data and records the starting byte offset of each line.
func BuildLineOffsets(data []byte) Offset {
	offsets := []LineOffset{{Line: 1, Start: 0}}
	line := 1
	for i, b := range data {
		if b == '\n' {
			line++
			offsets = append(offsets, LineOffset{Line: line, Start: int64(i + 1)})
		}
	}
	return Offset{offsets: offsets}
}

// GetPosition maps a byte offset to its corresponding line and column numbers.
func (o Offset) GetPosition(offset int64) dyn.Location {
	// Binary search to find the line
	idx := sort.Search(len(o.offsets), func(i int) bool {
		return o.offsets[i].Start > offset
	}) - 1

	if idx < 0 {
		idx = 0
	}

	lineOffset := o.offsets[idx]
	return dyn.Location{
		File:   o.source,
		Line:   lineOffset.Line,
		Column: int(offset-lineOffset.Start) + 1,
	}
}

func (o *Offset) SetSource(source string) {
	o.source = source
}
