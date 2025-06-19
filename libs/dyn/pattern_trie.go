package dyn

import "fmt"

// PatternTrie is a trie data structure for storing and querying patterns.
// It supports both exact matches and wildcard matches. You can insert [Pattern]s
// into the trie and then query it to see if a given [Path] matches any of the
// patterns.
type PatternTrie struct {
	root *trieNode
}

// trieNode represents a node in the pattern trie.
// Each node in the array represents one or more of:
// 1. An [AnyKey] component. This is the "*" wildcard which matches any map key.
// 2. An [AnyIndex] component. This is the "[*]" wildcard which matches any array index.
// 3. Multiple [Key] components. These are multiple static path keys for this this node would match.
// 4. Multiple [Index] components. These are multiple static path indices for this this node would match.
//
// Note: It's valid for both anyKey and pathKey to be set at the same time.
// For example, adding both "foo.*.bar" and "foo.bar" to a trie is valid.
// Similarly, it's valid for both anyIndex and pathIndex to be set at the same time.
// For example, adding both "foo[*].bar" and "foo[0]" to a trie is valid.
//
// Note: Setting both key (one of pathKey or anyKey) and index (one of pathIndex or anyIndex)
// is not supported by the [PatternTrie.SearchPath] method. We don't perform validation for this
// case because it's not expected to arise in practice where a field is either a map or an array,
// but not both.
type trieNode struct {
	// If set this indicates the trie node is an anyKey node.
	// Maps to the [AnyKey] component.
	anyKey *trieNode

	// Indicates the trie node is an anyIndex node.
	// Maps to the [AnyIndex] component.
	anyIndex *trieNode

	// Set of strings which this trie node matches.
	// Maps to the [Key] component.
	pathKey map[string]*trieNode

	// Set of indices which this trie node matches.
	// Maps to the [Index] component.
	pathIndex map[int]*trieNode

	// Indicates if this node is the end of a pattern. Encountering a node
	// with isEnd set to true in a trie means the pattern from the root to this
	// node is a complete pattern.
	isEnd bool
}

// NewPatternTrie creates a new empty pattern trie.
func NewPatternTrie() *PatternTrie {
	return &PatternTrie{
		root: &trieNode{},
	}
}

// Insert adds a pattern to the trie.
func (t *PatternTrie) Insert(pattern Pattern) error {
	// Empty pattern represents the root.
	if len(pattern) == 0 {
		t.root.isEnd = true
		return nil
	}

	current := t.root
	for i, component := range pattern {
		// Create next node based on component type
		var next *trieNode
		switch c := component.(type) {
		case anyKeyComponent:
			if current.anyKey == nil {
				current.anyKey = &trieNode{}
			}
			next = current.anyKey

		case anyIndexComponent:
			if current.anyIndex == nil {
				current.anyIndex = &trieNode{}
			}
			next = current.anyIndex

		case pathComponent:
			if key := c.Key(); key != "" {
				if current.pathKey == nil {
					current.pathKey = make(map[string]*trieNode)
				}
				if _, exists := current.pathKey[key]; !exists {
					current.pathKey[key] = &trieNode{}
				}
				next = current.pathKey[key]
			} else {
				idx := c.Index()
				if current.pathIndex == nil {
					current.pathIndex = make(map[int]*trieNode)
				}
				if _, exists := current.pathIndex[idx]; !exists {
					current.pathIndex[idx] = &trieNode{}
				}
				next = current.pathIndex[idx]
			}
		}

		if next == nil {
			return fmt.Errorf("invalid component type: %T", component)
		}

		// Mark as end of pattern if this is the last component.
		if !next.isEnd && i == len(pattern)-1 {
			next.isEnd = true
		}

		// Move to next node
		current = next
	}

	return nil
}

// SearchPath checks if the given path matches any pattern in the trie.
// A path matches if it exactly matches a pattern or if it matches a pattern with wildcards.
func (t *PatternTrie) SearchPath(path Path) (Pattern, bool) {
	// We statically allocate the prefix array that is used to track the current
	// prefix accumulated while walking the prefix tree. Having the static allocation
	// ensures that we do not allocate memory on every recursive call.
	prefix := make(Pattern, len(path))
	pattern, ok := t.searchPathRecursive(t.root, path, prefix, 0)
	return pattern, ok
}

// searchPathRecursive is a helper function that recursively checks if a path matches any pattern.
// Arguments:
// - node: the current node in the trie.
// - path: the path to check.
// - prefix: the prefix accumulated while walking the prefix tree.
// - index: the current index in the path / prefix
//
// Note we always expect the path and prefix to be the same length because wildcards like * and [*]
// only match a single path component.
func (t *PatternTrie) searchPathRecursive(node *trieNode, path Path, prefix Pattern, index int) (Pattern, bool) {
	if node == nil {
		return nil, false
	}

	// Zero case, when the query path is the root node. We return nil here to match
	// the semantics of [MustPatternFromString] which returns nil for the empty string.
	if len(path) == 0 {
		return nil, node.isEnd
	}

	// If we've reached the end of the path, check if this node is a valid end of a pattern.
	isLast := index == len(path)
	if isLast {
		return prefix, node.isEnd
	}

	currentComponent := path[index]

	// First check if the key wildcard is set for the current index.
	if currentComponent.isKey() && node.anyKey != nil {
		prefix[index] = AnyKey()
		pattern, ok := t.searchPathRecursive(node.anyKey, path, prefix, index+1)
		if ok {
			return pattern, true
		}
	}

	// If no key wildcard is set, check if the key is an exact match.
	if currentComponent.isKey() {
		child, exists := node.pathKey[currentComponent.Key()]
		if !exists {
			return prefix, false
		}
		prefix[index] = currentComponent
		return t.searchPathRecursive(child, path, prefix, index+1)
	}

	// First check if the index wildcard is set for the current index.
	if currentComponent.isIndex() && node.anyIndex != nil {
		prefix[index] = AnyIndex()
		pattern, ok := t.searchPathRecursive(node.anyIndex, path, prefix, index+1)
		if ok {
			return pattern, true
		}
	}

	// If no index wildcard is set, check if the index is an exact match.
	if currentComponent.isIndex() {
		child, exists := node.pathIndex[currentComponent.Index()]
		if !exists {
			return prefix, false
		}
		prefix[index] = currentComponent
		return t.searchPathRecursive(child, path, prefix, index+1)
	}

	// If we've reached this point, the path does not match any patterns in the trie.
	return prefix, false
}
