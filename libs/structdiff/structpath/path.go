package structpath

import (
	"fmt"
	"strconv"

	"github.com/databricks/cli/libs/structdiff/jsontag"
)

// PathNode represents a node in a path for struct diffing.
// It can represent struct fields, map keys, or array/slice indices.
type PathNode struct {
	prev    *PathNode
	jsonTag jsontag.JSONTag // For lazy JSON key resolution
	key     string          // Computed key (JSON key for structs, string key for maps, or Go field name for fallback)
	// If index >= 0, the node specifies a slice/array index in index.
	// If index == -1, the node specifies a struct attribute
	// If index == -2, the node specifies a map key in key
	// If index == -3, the node specifies an unresolved struct attribute
	index int
}

// NewIndex creates a new PathNode for an array/slice index.
func NewIndex(prev *PathNode, index int) *PathNode {
	return &PathNode{
		prev:  prev,
		index: index,
	}
}

// NewMapKey creates a new PathNode for a map key.
func NewMapKey(prev *PathNode, key string) *PathNode {
	return &PathNode{
		prev:  prev,
		key:   key,
		index: -2,
	}
}

// NewStructField creates a new PathNode for a struct field.
// The jsonTag is used for lazy JSON key resolution, and fieldName is used as fallback.
func NewStructField(prev *PathNode, jsonTag jsontag.JSONTag, fieldName string) *PathNode {
	return &PathNode{
		prev:    prev,
		jsonTag: jsonTag,
		key:     fieldName,
		index:   -3, // Unresolved struct attribute
	}
}

// String returns the string representation of the path.
func (p *PathNode) String() string {
	if p == nil {
		return ""
	}

	if p.index >= 0 {
		return p.prev.String() + "[" + strconv.Itoa(p.index) + "]"
	}

	if p.index == -3 {
		// Lazy resolve JSON key for struct fields
		jsonName := p.jsonTag.Name()
		if jsonName != "" {
			p.key = jsonName
		}
		// If jsonName is empty, key already contains the Go field name as fallback
		p.index = -1
	}

	if p.index == -1 {
		return p.prev.String() + "." + p.key
	}

	return fmt.Sprintf("%s[%q]", p.prev.String(), p.key)
}
