package structpath

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	tagStruct      = -1
	tagMapKey      = -2
	tagDotStar     = -4
	tagBracketStar = -5
)

// PathNode represents a node in a path for struct diffing.
// It can represent struct fields, map keys, or array/slice indices.
type PathNode struct {
	prev *PathNode
	key  string // Computed key (JSON key for structs, string key for maps, or Go field name for fallback)
	// If index >= 0, the node specifies a slice/array index in index.
	// If index < 0, this describes the type of node (see tagStruct and other consts above)
	index int
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

func (p *PathNode) DotStar() bool {
	if p == nil {
		return false
	}
	return p.index == tagDotStar
}

func (p *PathNode) BracketStar() bool {
	if p == nil {
		return false
	}
	return p.index == tagBracketStar
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

// AsSlice returns the path as a slice of PathNodes from root to current.
// Efficiently pre-allocates the exact length and fills in reverse order.
func (p *PathNode) AsSlice() []*PathNode {
	// First pass: count the length
	length := 0
	current := p
	for current != nil {
		length++
		current = current.Parent()
	}

	// Allocate slice with exact capacity
	segments := make([]*PathNode, length)

	// Second pass: fill in reverse order (from end to start)
	current = p
	for i := length - 1; i >= 0; i-- {
		segments[i] = current
		current = current.Parent()
	}

	return segments
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
// The fieldName should be the resolved field name (e.g., from JSON tag or Go field name).
func NewStructField(prev *PathNode, fieldName string) *PathNode {
	return &PathNode{
		prev:  prev,
		key:   fieldName,
		index: tagStruct,
	}
}

func NewDotStar(prev *PathNode) *PathNode {
	return &PathNode{
		prev:  prev,
		index: tagDotStar,
	}
}

func NewBracketStar(prev *PathNode) *PathNode {
	return &PathNode{
		prev:  prev,
		index: tagBracketStar,
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

	if p.index == tagDotStar {
		prev := p.prev.String()
		if prev == "" {
			return "*"
		}
		return prev + ".*"
	}

	if p.index == tagBracketStar {
		prev := p.prev.String()
		if prev == "" {
			return "[*]"
		}
		return prev + "[*]"
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
//   - START: Beginning of parsing, expects field name, "[", or "*"
//   - FIELD_START: After a dot, expects field name or "*"
//   - FIELD: Reading field name characters
//   - DOT_STAR: Encountered "*" (at start or after dot), expects ".", "[", or EOF
//   - BRACKET_OPEN: Just encountered "[", expects digit, "'" or "*"
//   - INDEX: Reading array index digits, expects more digits or "]"
//   - MAP_KEY: Reading map key content, expects any char or "'"
//   - MAP_KEY_QUOTE: Encountered "'" in map key, expects "'" (escape) or "]" (end)
//   - WILDCARD: Reading "*" in brackets, expects "]"
//   - EXPECT_DOT_OR_END: After bracket close, expects ".", "[" or end of string
//   - END: Successfully completed parsing
//
// Transitions:
//   - START: [a-zA-Z_-] -> FIELD, "[" -> BRACKET_OPEN, "*" -> DOT_STAR, EOF -> END
//   - FIELD_START: [a-zA-Z_-] -> FIELD, "*" -> DOT_STAR, other -> ERROR
//   - FIELD: [a-zA-Z0-9_-] -> FIELD, "." -> FIELD_START, "[" -> BRACKET_OPEN, EOF -> END
//   - DOT_STAR: "." -> FIELD_START, "[" -> BRACKET_OPEN, EOF -> END, other -> ERROR
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
		stateDotStar
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
			} else if ch == '*' {
				state = stateDotStar
			} else if !isReservedFieldChar(ch) {
				currentToken.WriteByte(ch)
				state = stateField
			} else {
				return nil, fmt.Errorf("unexpected character '%c' at position %d", ch, pos)
			}

		case stateFieldStart:
			if ch == '*' {
				state = stateDotStar
			} else if !isReservedFieldChar(ch) {
				currentToken.WriteByte(ch)
				state = stateField
			} else {
				return nil, fmt.Errorf("expected field name after '.' but got '%c' at position %d", ch, pos)
			}

		case stateField:
			if ch == '.' {
				result = NewStructField(result, currentToken.String())
				currentToken.Reset()
				state = stateFieldStart
			} else if ch == '[' {
				result = NewStructField(result, currentToken.String())
				currentToken.Reset()
				state = stateBracketOpen
			} else if !isReservedFieldChar(ch) {
				currentToken.WriteByte(ch)
			} else {
				return nil, fmt.Errorf("invalid character '%c' in field name at position %d", ch, pos)
			}

		case stateDotStar:
			switch ch {
			case '.':
				result = NewDotStar(result)
				state = stateFieldStart
			case '[':
				result = NewDotStar(result)
				state = stateBracketOpen
			default:
				return nil, fmt.Errorf("unexpected character '%c' after '.*' at position %d", ch, pos)
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
				result = NewBracketStar(result)
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
		result = NewStructField(result, currentToken.String())
		return result, nil
	case stateDotStar:
		result = NewDotStar(result)
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
