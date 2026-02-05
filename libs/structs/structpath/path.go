package structpath

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

const (
	// Encodes wildcard after a dot: foo.*
	tagDotStar = -2

	// Encodes wildcard in brackets: foo[*]
	tagBracketStar = -3

	// Encodes key/value index, which is encoded as [key='value'] or [key="value"]
	tagKeyValue = -4

	// Encodes .field syntax
	// Note, most users should use StringKey() method which represents both DotString and BracketString
	tagDotString = -5

	// Encodes ["field"] syntax
	tagBracketString = -6
)

type PathNodeBase struct {
	key string // Computed key (JSON key for structs, string key for maps, or Go field name for fallback)
	// If index >= 0, the node specifies a slice/array index in index.
	// If index < 0, this describes the type of node
	index int
	value string // Used for tagKeyValue: stores the value part of [key='value']
}

// PathNode represents a node in a path for struct diffing.
// It can represent struct fields, map keys, or array/slice indices.
type PathNode struct {
	PathNodeBase
	prev *PathNode
}

// PatternNode represents a node that can also support wildcards
type PatternNode struct {
	PathNodeBase
	prev *PatternNode
}

func (p *PathNode) IsRoot() bool {
	return p == nil
}

func (p *PatternNode) IsRoot() bool {
	return p == nil
}

func (p *PatternNode) Parent() *PatternNode {
	if p == nil {
		return nil
	}
	return p.prev
}

// Index - nil-safe wrappers
func (p *PathNode) Index() (int, bool) {
	if p == nil {
		return -1, false
	}
	return p.PathNodeBase.Index()
}

func (p *PatternNode) Index() (int, bool) {
	if p == nil {
		return -1, false
	}
	return p.PathNodeBase.Index()
}

func (p *PathNodeBase) Index() (int, bool) {
	if p.index >= 0 {
		return p.index, true
	}
	return -1, false
}

func (p *PathNodeBase) DotStar() bool {
	return p.index == tagDotStar
}

func (p *PathNodeBase) BracketStar() bool {
	return p.index == tagBracketStar
}

// KeyValue - nil-safe wrappers
func (p *PathNode) KeyValue() (key, value string, ok bool) {
	if p == nil {
		return "", "", false
	}
	return p.PathNodeBase.KeyValue()
}

func (p *PatternNode) KeyValue() (key, value string, ok bool) {
	if p == nil {
		return "", "", false
	}
	return p.PathNodeBase.KeyValue()
}

func (p *PathNodeBase) KeyValue() (key, value string, ok bool) {
	if p.index == tagKeyValue {
		return p.key, p.value, true
	}
	return "", "", false
}

// StringKey - nil-safe wrappers (required because Go panics accessing embedded field on nil receiver)
func (p *PathNode) StringKey() (string, bool) {
	if p == nil {
		return "", false
	}
	return p.PathNodeBase.StringKey()
}

func (p *PatternNode) StringKey() (string, bool) {
	if p == nil {
		return "", false
	}
	return p.PathNodeBase.StringKey()
}

func (p *PathNodeBase) IsDotString() bool {
	return p != nil && p.index == tagDotString
}

func (p *PathNodeBase) DotString() (string, bool) {
	if p == nil {
		return "", false
	}
	if p.index == tagDotString {
		return p.key, true
	}
	return "", false
}

func (p *PathNodeBase) BracketString() (string, bool) {
	if p == nil {
		return "", false
	}
	if p.index == tagBracketString {
		return p.key, true
	}
	return "", false
}

// formatNode writes the string representation of this node to the builder.
// isFirst indicates if this is the first node in the path (affects dot prefix).
func (p *PathNodeBase) formatNode(result *strings.Builder, isFirst bool) {
	if p.index >= 0 {
		result.WriteString("[")
		result.WriteString(strconv.Itoa(p.index))
		result.WriteString("]")
	} else if p.index == tagDotStar {
		if isFirst {
			result.WriteString("*")
		} else {
			result.WriteString(".*")
		}
	} else if p.index == tagBracketStar {
		result.WriteString("[*]")
	} else if p.index == tagKeyValue {
		result.WriteString("[")
		result.WriteString(p.key)
		result.WriteString("=")
		result.WriteString(EncodeMapKey(p.value))
		result.WriteString("]")
	} else if p.index == tagDotString {
		if !isFirst {
			result.WriteString(".")
		}
		result.WriteString(p.key)
	} else if p.index == tagBracketString {
		result.WriteString("[")
		result.WriteString(EncodeMapKey(p.key))
		result.WriteString("]")
	}
}

// StringKey returns either Field() or MapKey() if either is available
func (p *PathNodeBase) StringKey() (string, bool) {
	if p == nil {
		return "", false
	}
	if p.index == tagDotString || p.index == tagBracketString {
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
	length := p.Len()
	segments := make([]*PathNode, length)

	// Fill in reverse order
	current := p
	for i := length - 1; i >= 0; i-- {
		segments[i] = current
		current = current.Parent()
	}

	return segments
}

// AsSlice returns the pattern as a slice of PatternNodes from root to current.
func (p *PatternNode) AsSlice() []*PatternNode {
	length := p.Len()
	segments := make([]*PatternNode, length)

	// Fill in reverse order
	current := p
	for i := length - 1; i >= 0; i-- {
		segments[i] = current
		current = current.Parent()
	}

	return segments
}

// Len returns the number of components in the pattern.
func (p *PatternNode) Len() int {
	length := 0
	current := p
	for current != nil {
		length++
		current = current.Parent()
	}
	return length
}

// String returns the string representation of the pattern.
func (p *PatternNode) String() string {
	if p == nil {
		return ""
	}
	components := p.AsSlice()
	var result strings.Builder
	for i, node := range components {
		node.formatNode(&result, i == 0)
	}
	return result.String()
}

// NewIndex creates a new PathNode for an array/slice index.
func NewIndex(prev *PathNode, index int) *PathNode {
	if index < 0 {
		panic("index must be non-negative")
	}
	return &PathNode{
		PathNodeBase: PathNodeBase{index: index},
		prev:         prev,
	}
}

// NewDotString creates a PathNode for dot notation (.field).
func NewDotString(prev *PathNode, fieldName string) *PathNode {
	return &PathNode{
		PathNodeBase: PathNodeBase{key: fieldName, index: tagDotString},
		prev:         prev,
	}
}

// NewBracketString creates a PathNode for bracket notation (["field"]).
func NewBracketString(prev *PathNode, fieldName string) *PathNode {
	return &PathNode{
		PathNodeBase: PathNodeBase{key: fieldName, index: tagBracketString},
		prev:         prev,
	}
}

// NewStringKey creates a PathNode, choosing dot notation if the fieldName is a valid field name,
// otherwise bracket notation.
func NewStringKey(prev *PathNode, fieldName string) *PathNode {
	if isValidField(fieldName) {
		return NewDotString(prev, fieldName)
	}
	return NewBracketString(prev, fieldName)
}

func NewKeyValue(prev *PathNode, key, value string) *PathNode {
	return &PathNode{
		PathNodeBase: PathNodeBase{key: key, index: tagKeyValue, value: value},
		prev:         prev,
	}
}

// PatternNode constructors

// NewPatternIndex creates a new PatternNode for an array/slice index.
func NewPatternIndex(prev *PatternNode, index int) *PatternNode {
	if index < 0 {
		panic("index must be non-negative")
	}
	return &PatternNode{
		PathNodeBase: PathNodeBase{index: index},
		prev:         prev,
	}
}

// NewPatternDotString creates a PatternNode for dot notation (.field).
func NewPatternDotString(prev *PatternNode, fieldName string) *PatternNode {
	return &PatternNode{
		PathNodeBase: PathNodeBase{key: fieldName, index: tagDotString},
		prev:         prev,
	}
}

// NewPatternBracketString creates a PatternNode for bracket notation (["field"]).
func NewPatternBracketString(prev *PatternNode, fieldName string) *PatternNode {
	return &PatternNode{
		PathNodeBase: PathNodeBase{key: fieldName, index: tagBracketString},
		prev:         prev,
	}
}

// NewPatternStringKey creates a PatternNode, choosing dot notation if the fieldName is a valid field name,
// otherwise bracket notation.
func NewPatternStringKey(prev *PatternNode, fieldName string) *PatternNode {
	if isValidField(fieldName) {
		return NewPatternDotString(prev, fieldName)
	}
	return NewPatternBracketString(prev, fieldName)
}

func NewPatternDotStar(prev *PatternNode) *PatternNode {
	return &PatternNode{
		PathNodeBase: PathNodeBase{index: tagDotStar},
		prev:         prev,
	}
}

func NewPatternBracketStar(prev *PatternNode) *PatternNode {
	return &PatternNode{
		PathNodeBase: PathNodeBase{index: tagBracketStar},
		prev:         prev,
	}
}

func NewPatternKeyValue(prev *PatternNode, key, value string) *PatternNode {
	return &PatternNode{
		PathNodeBase: PathNodeBase{key: key, index: tagKeyValue, value: value},
		prev:         prev,
	}
}

// String returns the string representation of the path.
// The string keys are encoded in dot syntax (foo.bar) if they don't have any reserved characters (so can be parsed as fields).
// Otherwise they are encoded in brackets + single quotes: tags['name']. Single quote can escaped by placing two single quotes.
// This encoding is chosen over traditional double quotes because when encoded in JSON it does not need to be escaped:
//
//	{
//		"resources.jobs.foo.tags['cost-center']": {}
//	}
func (p *PathNode) String() string {
	if p == nil {
		return ""
	}
	components := p.AsSlice()
	var result strings.Builder
	for i, node := range components {
		node.formatNode(&result, i == 0)
	}
	return result.String()
}

func EncodeMapKey(s string) string {
	escaped := strings.ReplaceAll(s, "'", "''")
	return "'" + escaped + "'"
}

// Parse parses a string representation of a path using a state machine.
// If wildcardAllowed is true, returns (nil, *PatternNode, nil) on success.
// If wildcardAllowed is false, returns (*PathNode, nil, nil) on success but errors if wildcards are found.
//
// State Machine for Path Parsing:
//
// States:
//   - START: Beginning of parsing, expects field name, "[", or "*"
//   - FIELD_START: After a dot, expects field name or "*"
//   - FIELD: Reading field name characters
//   - DOT_STAR: Encountered "*" (at start or after dot), expects ".", "[", or EOF
//   - BRACKET_OPEN: Just encountered "[", expects digit, "'", "*", or identifier for key-value
//   - INDEX: Reading array index digits, expects more digits or "]"
//   - MAP_KEY: Reading map key content, expects any char or "'"
//   - MAP_KEY_QUOTE: Encountered "'" in map key, expects "'" (escape) or "]" (end)
//   - WILDCARD: Reading "*" in brackets, expects "]"
//   - KEYVALUE_KEY: Reading key part of [key='value'], expects identifier chars or "="
//   - KEYVALUE_EQUALS: After key, expecting "'" to start value
//   - KEYVALUE_VALUE: Reading value content, expects any char or quote
//   - KEYVALUE_VALUE_QUOTE: Encountered quote in value, expects same quote (escape) or "]" (end)
//   - EXPECT_DOT_OR_END: After bracket close, expects ".", "[" or end of string
//   - END: Successfully completed parsing
func Parse(s string, wildcardAllowed bool) (*PathNode, *PatternNode, error) {
	if s == "" {
		return nil, nil, nil
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
		stateKeyValueKey
		stateKeyValueEquals
		stateKeyValueValue
		stateKeyValueValueQuote
		stateExpectDotOrEnd
		stateEnd
	)

	// Parse into a slice of PathNodeBase values first
	var nodes []PathNodeBase
	state := stateStart
	var currentToken strings.Builder
	var keyValueKey string
	pos := 0
	hasWildcard := false

parseLoop:
	for pos < len(s) {
		ch := s[pos]

		switch state {
		case stateStart:
			if ch == '[' {
				state = stateBracketOpen
			} else if ch == '*' {
				hasWildcard = true
				state = stateDotStar
			} else if !isReservedFieldChar(ch) {
				currentToken.WriteByte(ch)
				state = stateField
			} else {
				return nil, nil, fmt.Errorf("unexpected character '%c' at position %d", ch, pos)
			}

		case stateFieldStart:
			if ch == '*' {
				hasWildcard = true
				state = stateDotStar
			} else if !isReservedFieldChar(ch) {
				currentToken.WriteByte(ch)
				state = stateField
			} else {
				return nil, nil, fmt.Errorf("expected field name after '.' but got '%c' at position %d", ch, pos)
			}

		case stateField:
			if ch == '.' {
				nodes = append(nodes, PathNodeBase{key: currentToken.String(), index: tagDotString})
				currentToken.Reset()
				state = stateFieldStart
			} else if ch == '[' {
				nodes = append(nodes, PathNodeBase{key: currentToken.String(), index: tagDotString})
				currentToken.Reset()
				state = stateBracketOpen
			} else if !isReservedFieldChar(ch) {
				currentToken.WriteByte(ch)
			} else {
				return nil, nil, fmt.Errorf("invalid character '%c' in field name at position %d", ch, pos)
			}

		case stateDotStar:
			switch ch {
			case '.':
				nodes = append(nodes, PathNodeBase{index: tagDotStar})
				state = stateFieldStart
			case '[':
				nodes = append(nodes, PathNodeBase{index: tagDotStar})
				state = stateBracketOpen
			default:
				return nil, nil, fmt.Errorf("unexpected character '%c' after '.*' at position %d", ch, pos)
			}

		case stateBracketOpen:
			if ch >= '0' && ch <= '9' {
				currentToken.WriteByte(ch)
				state = stateIndex
			} else if ch == '\'' {
				state = stateMapKey
			} else if ch == '*' {
				hasWildcard = true
				state = stateWildcard
			} else if !isReservedFieldChar(ch) {
				currentToken.WriteByte(ch)
				state = stateKeyValueKey
			} else {
				return nil, nil, fmt.Errorf("unexpected character '%c' after '[' at position %d", ch, pos)
			}

		case stateIndex:
			if ch >= '0' && ch <= '9' {
				currentToken.WriteByte(ch)
			} else if ch == ']' {
				index, err := strconv.Atoi(currentToken.String())
				if err != nil {
					return nil, nil, fmt.Errorf("invalid index '%s' at position %d", currentToken.String(), pos-len(currentToken.String()))
				}
				nodes = append(nodes, PathNodeBase{index: index})
				currentToken.Reset()
				state = stateExpectDotOrEnd
			} else {
				return nil, nil, fmt.Errorf("unexpected character '%c' in index at position %d", ch, pos)
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
				currentToken.WriteByte('\'')
				state = stateMapKey
			case ']':
				nodes = append(nodes, PathNodeBase{key: currentToken.String(), index: tagBracketString})
				currentToken.Reset()
				state = stateExpectDotOrEnd
			default:
				return nil, nil, fmt.Errorf("unexpected character '%c' after quote in map key at position %d", ch, pos)
			}

		case stateWildcard:
			if ch == ']' {
				nodes = append(nodes, PathNodeBase{index: tagBracketStar})
				state = stateExpectDotOrEnd
			} else {
				return nil, nil, fmt.Errorf("unexpected character '%c' after '*' at position %d", ch, pos)
			}

		case stateKeyValueKey:
			if ch == '=' {
				keyValueKey = currentToken.String()
				currentToken.Reset()
				state = stateKeyValueEquals
			} else if !isReservedFieldChar(ch) {
				currentToken.WriteByte(ch)
			} else {
				return nil, nil, fmt.Errorf("unexpected character '%c' in key-value key at position %d", ch, pos)
			}

		case stateKeyValueEquals:
			if ch == '\'' {
				state = stateKeyValueValue
			} else {
				return nil, nil, fmt.Errorf("expected quote after '=' but got '%c' at position %d", ch, pos)
			}

		case stateKeyValueValue:
			if ch == '\'' {
				state = stateKeyValueValueQuote
			} else {
				currentToken.WriteByte(ch)
			}

		case stateKeyValueValueQuote:
			switch ch {
			case '\'':
				currentToken.WriteByte(ch)
				state = stateKeyValueValue
			case ']':
				nodes = append(nodes, PathNodeBase{key: keyValueKey, index: tagKeyValue, value: currentToken.String()})
				currentToken.Reset()
				keyValueKey = ""
				state = stateExpectDotOrEnd
			default:
				return nil, nil, fmt.Errorf("unexpected character '%c' after quote in key-value at position %d", ch, pos)
			}

		case stateExpectDotOrEnd:
			switch ch {
			case '.':
				state = stateFieldStart
			case '[':
				state = stateBracketOpen
			default:
				return nil, nil, fmt.Errorf("unexpected character '%c' at position %d", ch, pos)
			}

		case stateEnd:
			break parseLoop

		default:
			return nil, nil, fmt.Errorf("parser error at position %d", pos)
		}

		pos++
	}

	// Handle end-of-input based on final state
	switch state {
	case stateStart:
		// Empty path
	case stateField:
		nodes = append(nodes, PathNodeBase{key: currentToken.String(), index: tagDotString})
	case stateDotStar:
		nodes = append(nodes, PathNodeBase{index: tagDotStar})
	case stateExpectDotOrEnd:
		// Already complete
	case stateFieldStart:
		return nil, nil, errors.New("unexpected end of input after '.'")
	case stateBracketOpen:
		return nil, nil, errors.New("unexpected end of input after '['")
	case stateIndex:
		return nil, nil, errors.New("unexpected end of input while parsing index")
	case stateMapKey:
		return nil, nil, errors.New("unexpected end of input while parsing map key")
	case stateMapKeyQuote:
		return nil, nil, errors.New("unexpected end of input after quote in map key")
	case stateWildcard:
		return nil, nil, errors.New("unexpected end of input after wildcard '*'")
	case stateKeyValueKey:
		return nil, nil, errors.New("unexpected end of input while parsing key-value key")
	case stateKeyValueEquals:
		return nil, nil, errors.New("unexpected end of input after '=' in key-value")
	case stateKeyValueValue:
		return nil, nil, errors.New("unexpected end of input while parsing key-value value")
	case stateKeyValueValueQuote:
		return nil, nil, errors.New("unexpected end of input after quote in key-value value")
	case stateEnd:
		// Already complete
	default:
		return nil, nil, fmt.Errorf("parser error at position %d", pos)
	}

	// Check wildcard constraint
	if hasWildcard && !wildcardAllowed {
		return nil, nil, errors.New("wildcards not allowed in path")
	}

	// Build the appropriate linked list
	if len(nodes) == 0 {
		return nil, nil, nil
	}

	if wildcardAllowed {
		// Build PatternNode chain
		var result *PatternNode
		for _, node := range nodes {
			result = &PatternNode{
				PathNodeBase: node,
				prev:         result,
			}
		}
		return nil, result, nil
	}

	// Build PathNode chain
	var result *PathNode
	for _, node := range nodes {
		result = &PathNode{
			PathNodeBase: node,
			prev:         result,
		}
	}
	return result, nil, nil
}

// ParsePath parses a path string. Wildcards are not allowed.
func ParsePath(s string) (*PathNode, error) {
	path, _, err := Parse(s, false)
	return path, err
}

// ParsePattern parses a pattern string. Wildcards are allowed.
func ParsePattern(s string) (*PatternNode, error) {
	_, pattern, err := Parse(s, true)
	return pattern, err
}

// MustParsePath parses a path string and panics on error. Wildcards are not allowed.
func MustParsePath(s string) *PathNode {
	path, err := ParsePath(s)
	if err != nil {
		panic(err)
	}
	return path
}

// MustParsePattern parses a pattern string and panics on error. Wildcards are allowed.
func MustParsePattern(s string) *PatternNode {
	pattern, err := ParsePattern(s)
	if err != nil {
		panic(err)
	}
	return pattern
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
	case '\'':
		return true
	case ' ':
		return true
	case '}':
		return true
	case '{':
		return true
	default:
		return false
	}
}

func isValidField(s string) bool {
	for ind := range s {
		if isReservedFieldChar(s[ind]) {
			return false
		}
	}
	return len(s) > 0
}

// PureReferenceToPath returns a PathNode if s is a pure variable reference, otherwise false.
// This function is similar to dynvar.PureReferenceToPath but returns a *PathNode instead of dyn.Path.
func PureReferenceToPath(s string) (*PathNode, bool) {
	ref, ok := dynvar.NewRef(dyn.V(s))
	if !ok {
		return nil, false
	}

	if !ref.IsPure() {
		return nil, false
	}

	pathNode, _, err := Parse(ref.References()[0], false)
	if err != nil {
		return nil, false
	}

	return pathNode, true
}

// SkipPrefix returns a new PathNode that skips the first n components of the path.
// If n is greater than or equal to the path length, returns nil (root).
func (p *PathNode) SkipPrefix(n int) *PathNode {
	if p.IsRoot() || n <= 0 {
		return p
	}

	length := p.Len()
	if n >= length {
		return nil // Return root
	}

	startNode := p.Prefix(n)

	var result *PathNode
	current := p
	for current != startNode {
		result = &PathNode{
			prev:         result,
			PathNodeBase: current.PathNodeBase,
			value: current.value,
		}
		current = current.Parent()
	}

	return result.ReverseInPlace()
}

// ReverseInPlace returns a new PathNode with the order of components reversed.
func (p *PathNode) ReverseInPlace() *PathNode {
	var result *PathNode
	current := p
	for current != nil {
		next := current.prev
		current.prev = result
		result = current
		current = next
	}
	return result
}

// Len returns the number of components in the path.
func (p *PathNode) Len() int {
	length := 0
	current := p
	for current != nil {
		length++
		current = current.Parent()
	}
	return length
}

// Prefix returns the PathNode at the nth position (1-indexed from root).
// If n is greater than the path length, returns the entire path.
// If n <= 0, returns nil (root).
func (p *PathNode) Prefix(n int) *PathNode {
	if p.IsRoot() || n <= 0 {
		return nil // Return root
	}

	// Find the path length first to handle edge cases
	length := p.Len()
	if n >= length {
		return p // Return entire path
	}

	// Traverse from root to find the nth node (1-indexed)
	current := p
	// Move to root first
	for range length - n {
		current = current.Parent()
	}

	return current
}

// HasPrefix tests whether the path begins with the given prefix PathNode.
// Returns true if all components of prefix match the first components of this path.
// A nil prefix (root) always returns true.
//
// Examples:
//   - "a.b".HasPrefix("a") returns true
//   - "a.b".HasPrefix("a.b.c") returns false (prefix is longer)
//   - "items[0].name".HasPrefix("items[0]") returns true
//   - "items[0].name".HasPrefix("items[1]") returns false (different index)
func (p *PathNode) HasPrefix(prefix *PathNode) bool {
	// Nil prefix (root) is always a prefix
	if prefix.IsRoot() {
		return true
	}

	// Nil path (root) can only have nil prefix
	if p.IsRoot() {
		return false
	}

	// Get lengths
	pLen := p.Len()
	prefixLen := prefix.Len()

	// Prefix cannot be longer than the path
	if prefixLen > pLen {
		return false
	}

	// Get to the position in p that corresponds to the last node of prefix
	pAtPrefixLen := p.Prefix(prefixLen)

	// Walk both paths backwards and compare each node
	currentP := pAtPrefixLen
	currentPrefix := prefix

	for currentPrefix != nil {
		if !nodesEqual(currentP, currentPrefix) {
			return false
		}
		currentP = currentP.prev
		currentPrefix = currentPrefix.prev
	}

	return true
}

// nodesEqual compares two PathNodes for equality (without comparing prev pointers).
func nodesEqual(a, b *PathNode) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.key == b.key && a.index == b.index && a.value == b.value
}

// MarshalYAML implements yaml.Marshaler for PathNode.
func (p *PathNode) MarshalYAML() (any, error) {
	return p.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler for PathNode.
// Note: wildcards are not allowed in PathNode; use PatternNode for paths with wildcards.
func (p *PathNode) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	parsed, _, err := Parse(s, false)
	if err != nil {
		return err
	}
	if parsed == nil {
		return nil
	}
	*p = *parsed
	return nil
}
