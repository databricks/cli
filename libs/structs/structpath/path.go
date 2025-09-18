package structpath

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/structs/structtag"
)

const (
	tagStruct   = -1
	tagMapKey   = -2
	tagAnyKey   = -4
	tagAnyIndex = -5
)

// PathNode represents a node in a path for struct diffing.
// It can represent struct fields, map keys, or array/slice indices.
type PathNode struct {
	prev      *PathNode
	jsonTag   structtag.JSONTag // For JSON key resolution
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

func (p *PathNode) Field() (string, bool) {
	if p == nil {
		return "", false
	}
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
// The jsonTag is used for JSON key resolution, and fieldName is used as fallback.
func NewStructField(prev *PathNode, tag reflect.StructTag, fieldName string) *PathNode {
	jsonTag := structtag.JSONTag(tag.Get("json"))
	bundleTag := structtag.BundleTag(tag.Get("bundle"))

	key := fieldName
	if name := jsonTag.Name(); name != "" {
		key = name
	}

	return &PathNode{
		prev:      prev,
		jsonTag:   jsonTag,
		bundleTag: bundleTag,
		key:       key,
		index:     tagStruct,
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
// The map keys are encoded in single quotes: tags['name']. Single quote can escaped by placing two single quotes: tags[””] (map key is one single quote).
// This encoding is chosen over traditional double quotes because when encoded in JSON it does not need to be escaped:
//
//	{
//		"resources.jobs.foo.tags['cost-center']": {}
//	}
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

	if p.index == tagStruct {
		prev := p.prev.String()
		if prev == "" {
			return p.key
		}
		return prev + "." + p.key
	}

	// Format map key with single quotes, escaping single quotes by doubling them
	escapedKey := strings.ReplaceAll(p.key, "'", "''")
	return fmt.Sprintf("%s['%s']", p.prev.String(), escapedKey)
}

// Parse parses a string representation of a path using a state machine.
//
// State Machine for Path Parsing:
//
// States:
//   - START: Beginning of parsing, expects field name or "["
//   - FIELD_START: After a dot, expects field name only
//   - FIELD: Reading field name characters
//   - BRACKET_OPEN: Just encountered "[", expects digit, "'" or "*"
//   - INDEX: Reading array index digits, expects more digits or "]"
//   - MAP_KEY: Reading map key content, expects any char or "'"
//   - MAP_KEY_QUOTE: Encountered "'" in map key, expects "'" (escape) or "]" (end)
//   - WILDCARD: Reading "*" in brackets, expects "]"
//   - EXPECT_DOT_OR_END: After bracket close, expects ".", "[" or end of string
//   - END: Successfully completed parsing
//
// Transitions:
//   - START: [a-zA-Z_-] -> FIELD, "[" -> BRACKET_OPEN, EOF -> END
//   - FIELD_START: [a-zA-Z_-] -> FIELD, other -> ERROR
//   - FIELD: [a-zA-Z0-9_-] -> FIELD, "." -> FIELD_START, "[" -> BRACKET_OPEN, EOF -> END
//   - BRACKET_OPEN: [0-9] -> INDEX, "'" -> MAP_KEY, "*" -> WILDCARD
//   - INDEX: [0-9] -> INDEX, "]" -> EXPECT_DOT_OR_END
//   - MAP_KEY: (any except "'") -> MAP_KEY, "'" -> MAP_KEY_QUOTE
//   - MAP_KEY_QUOTE: "'" -> MAP_KEY (escape), "]" -> EXPECT_DOT_OR_END (end key)
//   - WILDCARD: "]" -> EXPECT_DOT_OR_END
//   - EXPECT_DOT_OR_END: "." -> FIELD_START, "[" -> BRACKET_OPEN, EOF -> END
func Parse(s string) (*PathNode, error) {
	if s == "" {
		return nil, nil
	}

	// State machine states
	const (
		stateStart = iota
		stateFieldStart
		stateField
		stateBracketOpen
		stateIndex
		stateMapKey
		stateMapKeyQuote
		stateWildcard
		stateExpectDotOrEnd
		stateEnd
	)

	state := stateStart
	var result *PathNode
	var currentToken strings.Builder
	pos := 0

	for pos < len(s) {
		ch := s[pos]

		switch state {
		case stateStart:
			if ch == '[' {
				state = stateBracketOpen
			} else if !isReservedFieldChar(ch) {
				currentToken.WriteByte(ch)
				state = stateField
			} else {
				return nil, fmt.Errorf("unexpected character '%c' at position %d", ch, pos)
			}

		case stateFieldStart:
			if !isReservedFieldChar(ch) {
				currentToken.WriteByte(ch)
				state = stateField
			} else {
				return nil, fmt.Errorf("expected field name after '.' but got '%c' at position %d", ch, pos)
			}

		case stateField:
			if ch == '.' {
				result = NewStructField(result, reflect.StructTag(""), currentToken.String())
				currentToken.Reset()
				state = stateFieldStart
			} else if ch == '[' {
				result = NewStructField(result, reflect.StructTag(""), currentToken.String())
				currentToken.Reset()
				state = stateBracketOpen
			} else if !isReservedFieldChar(ch) {
				currentToken.WriteByte(ch)
			} else {
				return nil, fmt.Errorf("invalid character '%c' in field name at position %d", ch, pos)
			}

		case stateBracketOpen:
			if ch >= '0' && ch <= '9' {
				currentToken.WriteByte(ch)
				state = stateIndex
			} else if ch == '\'' {
				state = stateMapKey
			} else if ch == '*' {
				state = stateWildcard
			} else {
				return nil, fmt.Errorf("unexpected character '%c' after '[' at position %d", ch, pos)
			}

		case stateIndex:
			if ch >= '0' && ch <= '9' {
				currentToken.WriteByte(ch)
			} else if ch == ']' {
				index, err := strconv.Atoi(currentToken.String())
				if err != nil {
					return nil, fmt.Errorf("invalid index '%s' at position %d", currentToken.String(), pos-len(currentToken.String()))
				}
				result = NewIndex(result, index)
				currentToken.Reset()
				state = stateExpectDotOrEnd
			} else {
				return nil, fmt.Errorf("unexpected character '%c' in index at position %d", ch, pos)
			}

		case stateMapKey:
			switch ch {
			case '\'':
				state = stateMapKeyQuote
			default:
				currentToken.WriteByte(ch)
			}

		case stateMapKeyQuote:
			switch ch {
			case '\'':
				// Escaped quote - add single quote to key and continue
				currentToken.WriteByte('\'')
				state = stateMapKey
			case ']':
				// End of map key
				result = NewMapKey(result, currentToken.String())
				currentToken.Reset()
				state = stateExpectDotOrEnd
			default:
				return nil, fmt.Errorf("unexpected character '%c' after quote in map key at position %d", ch, pos)
			}

		case stateWildcard:
			if ch == ']' {
				// Note, since we're parsing this without type info present, we don't know if it's AnyKey or AnyIndex
				// Perhaps structpath should be simplified to have Wildcard as merged representation of AnyKey/AnyIndex
				result = NewAnyKey(result)
				state = stateExpectDotOrEnd
			} else {
				return nil, fmt.Errorf("unexpected character '%c' after '*' at position %d", ch, pos)
			}

		case stateExpectDotOrEnd:
			switch ch {
			case '.':
				state = stateFieldStart
			case '[':
				state = stateBracketOpen
			default:
				return nil, fmt.Errorf("unexpected character '%c' at position %d", ch, pos)
			}

		case stateEnd:
			return result, nil

		default:
			return nil, fmt.Errorf("parser error at position %d", pos)
		}

		pos++
	}

	// Handle end-of-input based on final state
	switch state {
	case stateStart:
		return result, nil // Empty path, result is nil
	case stateField:
		result = NewStructField(result, reflect.StructTag(""), currentToken.String())
		return result, nil
	case stateExpectDotOrEnd:
		return result, nil
	case stateFieldStart:
		return nil, errors.New("unexpected end of input after '.'")
	case stateBracketOpen:
		return nil, errors.New("unexpected end of input after '['")
	case stateIndex:
		return nil, errors.New("unexpected end of input while parsing index")
	case stateMapKey:
		return nil, errors.New("unexpected end of input while parsing map key")
	case stateMapKeyQuote:
		return nil, errors.New("unexpected end of input after quote in map key")
	case stateWildcard:
		return nil, errors.New("unexpected end of input after wildcard '*'")
	case stateEnd:
		return result, nil
	default:
		return nil, fmt.Errorf("parser error at position %d", pos)
	}
}

// isReservedFieldChar checks if character is reserved and cannot be used in field names
func isReservedFieldChar(ch byte) bool {
	switch ch {
	case ',': // Cannot appear in Golang JSON struct tag
		return true
	case '"': // Cannot appear in Golang struct tag
		return true
	case '`': // Cannot appear in Golang struct tag
		return true
	case '.': // Path separator
		return true
	case '[': // Bracket notation start
		return true
	case ']': // Bracket notation end
		return true
	default:
		return false
	}
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

	// Both struct fields and map keys use dot notation in DynPath
	prev := p.prev.DynPath()
	if prev == "" {
		return p.key
	} else {
		return prev + "." + p.key
	}
}
