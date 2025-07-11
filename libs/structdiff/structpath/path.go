package structpath

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/databricks/cli/libs/structdiff/structtag"
)

const (
	tagStruct           = -1
	tagMapKey           = -2
	tagUnresolvedStruct = -3
	tagAnyKey           = -4
	tagAnyIndex         = -5
)

// PathNode represents a node in a path for struct diffing.
// It can represent struct fields, map keys, or array/slice indices.
type PathNode struct {
	prev      *PathNode
	jsonTag   structtag.JSONTag // For lazy JSON key resolution
	bundleTag structtag.BundleTag
	key       string // Computed key (JSON key for structs, string key for maps, or Go field name for fallback)
	// If index >= 0, the node specifies a slice/array index in index.
	// If index < 0, this describes the type of node (see tagStruct and other consts above)
	index int
}

func (p *PathNode) JSONTag() structtag.JSONTag {
	return p.jsonTag
}

func (p *PathNode) BundleTag() structtag.BundleTag {
	return p.bundleTag
}

func (p *PathNode) IsRoot() bool {
	return p == nil
}

func (p *PathNode) Index() (int, bool) {
	if p == nil {
		return -1, false
	}
	if p.index >= 0 {
		return p.index, true
	}
	return -1, false
}

func (p *PathNode) MapKey() (string, bool) {
	if p == nil {
		return "", false
	}
	if p.index == tagMapKey {
		return p.key, true
	}
	return "", false
}

func (p *PathNode) AnyKey() bool {
	if p == nil {
		return false
	}
	return p.index == tagAnyKey
}

func (p *PathNode) AnyIndex() bool {
	if p == nil {
		return false
	}
	return p.index == tagAnyIndex
}

func (p *PathNode) resolveField() {
	if p.index == tagUnresolvedStruct {
		// Lazy resolve JSON key for struct fields
		jsonName := p.jsonTag.Name()
		if jsonName != "" {
			p.key = jsonName
		}
		// If jsonName is empty, key already contains the Go field name as fallback
		p.index = tagStruct
	}
}

func (p *PathNode) Field() (string, bool) {
	if p == nil {
		return "", false
	}
	p.resolveField()
	if p.index == tagStruct {
		return p.key, true
	}
	return "", false
}

func (p *PathNode) Parent() *PathNode {
	if p == nil {
		return nil
	}
	return p.prev
}

// NewIndex creates a new PathNode for an array/slice index.
func NewIndex(prev *PathNode, index int) *PathNode {
	if index < 0 {
		panic("index msut be non-negative")
	}
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
		index: tagMapKey,
	}
}

// NewStructField creates a new PathNode for a struct field.
// The jsonTag is used for lazy JSON key resolution, and fieldName is used as fallback.
func NewStructField(prev *PathNode, tag reflect.StructTag, fieldName string) *PathNode {
	jsonTag := structtag.JSONTag(tag.Get("json"))
	bundleTag := structtag.BundleTag(tag.Get("bundle"))

	return &PathNode{
		prev:      prev,
		jsonTag:   jsonTag,
		bundleTag: bundleTag,
		key:       fieldName,
		index:     tagUnresolvedStruct,
	}
}

func NewAnyKey(prev *PathNode) *PathNode {
	return &PathNode{
		prev:  prev,
		index: tagAnyKey,
	}
}

func NewAnyIndex(prev *PathNode) *PathNode {
	return &PathNode{
		prev:  prev,
		index: tagAnyIndex,
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

	if p.index == tagAnyKey || p.index == tagAnyIndex {
		return p.prev.String() + "[*]"
	}

	p.resolveField()

	if p.index == tagStruct {
		return p.prev.String() + "." + p.key
	}

	return fmt.Sprintf("%s[%q]", p.prev.String(), p.key)
}

// Path in libs/dyn format
func (p *PathNode) DynPath() string {
	if p == nil {
		return ""
	}

	if p.index >= 0 {
		return p.prev.DynPath() + "[" + strconv.Itoa(p.index) + "]"
	}

	if p.index == tagAnyKey {
		prev := p.prev.DynPath()
		if prev == "" {
			return "*"
		} else {
			return prev + ".*"
		}
	}

	if p.index == tagAnyIndex {
		return p.prev.DynPath() + "[*]"
	}

	p.resolveField()

	prev := p.prev.DynPath()
	if prev == "" {
		return p.key
	} else {
		return prev + "." + p.key
	}
}
