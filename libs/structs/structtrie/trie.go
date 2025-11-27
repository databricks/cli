package structtrie

import (
	"fmt"

	"github.com/databricks/cli/libs/structs/structpath"
)

/*

Prefix tree stores a map of path pattern to values.

Both concrete paths and patterns can be matched against the tree to find the matching value.

In case of multiple matches, longest and most concrete match wins.

Tree can be construct from map[string]any but internal format is more efficient.

Input:

{"*": "star",
  "grants": "grants slice",
  "grants[*]": "grant",
  "grants[*].principal": "principal",
}

Would match:
 "some_path" => "star",
 "grants" => "grants slice",
 "some_path.no_key" => nil (no match),
 "grants[0].principal" => "principal",
 "grants[1].principal" => "principal",
 "grants[*].principal" => "principal",

Prefix tree also supports efficient iteration, going to child and parent nodes.

Parent(): return parent node or nil for root
Child(node *PathNode) return child node in the tree best match matching node. Note, node.Parent() is not used here, only PathNode itself.


*/

// PrefixTree stores path patterns and associated values in a trie-like structure.
// Matches prefer the deepest node, with concrete segments taking precedence over wildcards.
type PrefixTree struct {
	Root *Node
}

// Node represents a single step inside the prefix tree.
// It keeps track of the component that leads to it, its value and the children below it.
type Node struct {
	parent    *Node
	component componentKey
	children  map[componentKey]*Node
	value     any
}

// NewPrefixTree returns an empty prefix tree with a root node.
func NewPrefixTree() *PrefixTree {
	return &PrefixTree{Root: newNode(componentKey{}, nil)}
}

// NewPrefixTreeFromMap constructs a prefix tree from serialized path patterns.
func NewPrefixTreeFromMap(values map[string]any) (*PrefixTree, error) {
	tree := NewPrefixTree()
	for raw, v := range values {
		path, err := structpath.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("parse %q: %w", raw, err)
		}
		if _, err = tree.Insert(path, v); err != nil {
			return nil, fmt.Errorf("insert %q: %w", raw, err)
		}
	}
	return tree, nil
}

// Insert adds or updates a value for the given path pattern.
// A nil path represents the root node.
func (t *PrefixTree) Insert(path *structpath.PathNode, value any) (*Node, error) {
	if t.Root == nil {
		t.Root = newNode(componentKey{}, nil)
	}

	if path == nil {
		if t.Root.value != nil {
			return nil, fmt.Errorf("path %q already exists", "")
		}
		t.Root.value = value
		return t.Root, nil
	}

	current := t.Root
	for _, segment := range path.AsSlice() {
		key, err := componentFromPattern(segment)
		if err != nil {
			return nil, err
		}
		if current.children == nil {
			current.children = make(map[componentKey]*Node)
		}
		child, exists := current.children[key]
		if !exists {
			child = newNode(key, current)
			current.children[key] = child
		}
		current = child
	}

	if current.value != nil {
		return nil, fmt.Errorf("path %q already exists", path.String())
	}

	current.value = value
	return current, nil
}

// InsertString parses the string path pattern and inserts the value.
func (t *PrefixTree) InsertString(path string, value any) (*Node, error) {
	parsed, err := structpath.Parse(path)
	if err != nil {
		return nil, err
	}
	return t.Insert(parsed, value)
}

// Match returns the node with the best matching value for the provided path.
// Matches prefer the deepest node. When depth ties, the node that used fewer wildcards wins.
func (t *PrefixTree) Match(path *structpath.PathNode) (*Node, bool) {
	if t == nil || t.Root == nil {
		return nil, false
	}

	if path == nil {
		if t.Root.value != nil {
			return t.Root, true
		}
		return nil, false
	}

	segments := path.AsSlice()
	var best matchResult
	t.match(t.Root, segments, 0, 0, 0, &best)

	if best.node != nil {
		return best.node, true
	}

	return nil, false
}

// MatchString parses the given path string and matches it against the tree.
func (t *PrefixTree) MatchString(path string) (*Node, bool, error) {
	parsed, err := structpath.Parse(path)
	if err != nil {
		return nil, false, err
	}
	node, ok := t.Match(parsed)
	return node, ok, nil
}

// Parent returns the parent node or nil for the root.
func (n *Node) Parent() *Node {
	if n == nil {
		return nil
	}
	return n.parent
}

// Child returns the child node that best matches the provided PathNode.
// Exact matches are preferred. When unavailable, wildcard children are returned.
func (n *Node) Child(pathNode *structpath.PathNode) *Node {
	if n == nil || pathNode == nil {
		return nil
	}

	if pathNode.DotStar() || pathNode.BracketStar() {
		return n.childFor(wildcardComponent)
	}

	if key, ok := pathNode.StringKey(); ok {
		if child := n.childFor(componentKey{kind: componentKindExact, key: key}); child != nil {
			return child
		}
	}

	return n.childFor(wildcardComponent)
}

// Children returns all child nodes.
func (n *Node) Children() []*Node {
	if n == nil || len(n.children) == 0 {
		return nil
	}

	result := make([]*Node, 0, len(n.children))
	for _, child := range n.children {
		result = append(result, child)
	}
	return result
}

// Value returns the stored value.
func (n *Node) Value() any {
	if n == nil {
		return nil
	}
	return n.value
}

/*
// SetValue overwrites the stored value.
func (n *Node) SetValue(value any) {
	if n == nil {
		return
	}
	n.value = value
}

// HasValue indicates whether the node stores a non-nil value.
func (n *Node) HasValue() bool {
	if n == nil {
		return false
	}
	return n.value != nil
}*/
/*
// Path returns the full path from the root to this node.
func (n *Node) Path() *structpath.PathNode {
	if n == nil || n.parent == nil {
		return nil
	}
	return n.component.append(n.parent.Path())
}*/

func (n *Node) childFor(key componentKey) *Node {
	if n == nil || len(n.children) == 0 {
		return nil
	}
	return n.children[key]
}

func newNode(component componentKey, parent *Node) *Node {
	return &Node{
		parent:    parent,
		component: component,
	}
}

type componentKind uint8

const (
	componentKindInvalid componentKind = iota
	componentKindExact
	componentKindWildcard
)

type componentKey struct {
	kind componentKind
	key  string
}

var wildcardComponent = componentKey{kind: componentKindWildcard}

func componentFromPattern(node *structpath.PathNode) (componentKey, error) {
	if node == nil {
		return componentKey{}, fmt.Errorf("nil path node")
	}

	if node.DotStar() || node.BracketStar() {
		return wildcardComponent, nil
	}

	if _, ok := node.Index(); ok {
		return componentKey{}, fmt.Errorf("array indexes are not supported in prefix tree keys")
	}

	if _, _, ok := node.KeyValue(); ok {
		return componentKey{}, fmt.Errorf("key-value selectors are not supported in prefix tree keys")
	}

	if key, ok := node.StringKey(); ok {
		return componentKey{
			kind: componentKindExact,
			key:  key,
		}, nil
	}

	return componentKey{}, fmt.Errorf("unsupported prefix tree component %q", node.String())
}

/*
func (c componentKey) append(prev *structpath.PathNode) *structpath.PathNode {
	switch c.kind {
	case componentKindExact:
		return structpath.NewStringKey(prev, c.key)
	case componentKindWildcard:
		return structpath.NewDotStar(prev)
	default:
		return prev
	}
}*/

func (t *PrefixTree) match(current *Node, segments []*structpath.PathNode, index, depth, concreteness int, best *matchResult) {
	if current == nil {
		return
	}

	if index == len(segments) {
		best.consider(current, depth, concreteness)
		return
	}

	children := current.matchingChildren(segments[index])
	for _, child := range children {
		nextConcreteness := concreteness
		if child.component.kind != componentKindWildcard {
			nextConcreteness++
		}
		t.match(child, segments, index+1, depth+1, nextConcreteness, best)
	}
}

func (n *Node) matchingChildren(pathNode *structpath.PathNode) []*Node {
	if len(n.children) == 0 {
		return nil
	}

	if pathNode == nil {
		return nil
	}

	if pathNode.DotStar() || pathNode.BracketStar() {
		if child, exists := n.children[wildcardComponent]; exists {
			return []*Node{child}
		}
		return nil
	}

	var out []*Node
	if key, ok := pathNode.StringKey(); ok {
		if child, exists := n.children[componentKey{kind: componentKindExact, key: key}]; exists {
			out = append(out, child)
		}
	}
	if child, exists := n.children[wildcardComponent]; exists {
		out = append(out, child)
	}
	return out
}

type matchResult struct {
	node         *Node
	depth        int
	concreteness int
}

func (m *matchResult) consider(node *Node, depth, concreteness int) {
	if node == nil || node.value == nil {
		return
	}
	if m.node == nil ||
		depth > m.depth ||
		(depth == m.depth && concreteness > m.concreteness) {
		m.node = node
		m.depth = depth
		m.concreteness = concreteness
	}
}
