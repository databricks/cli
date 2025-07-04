package dyn

import (
	"fmt"
)

// TrieNode is a trie data structure for storing and querying patterns.
// It supports both exact matches and wildcard matches. You can insert [Pattern]s
// into the trie and then query it to see if a given [Path] matches any of the
// patterns.
//
// TrieNode represents a node in the pattern trie.
// Each node in the array represents one or more of:
// 1. An [AnyKey] component. This is the "*" wildcard which matches any map key.
// 2. An [AnyIndex] component. This is the "[*]" wildcard which matches any array index.
// 3. Multiple [Key] components. These are multiple static path keys for this this node would match.
//
// Note: It's valid for both anyKey and pathKey to be set at the same time.
// For example, adding both "foo.*.bar" and "foo.bar" to a trie is valid.
//
// Note: Setting both key (one of pathKey or anyKey) and index (anyIndex)
// is not supported by the [PatternTrie.SearchPath] method. We don't perform validation for this
// case because it's not expected to arise in practice where a field is either a map or an array,
// but not both.
type TrieNode struct {
	// If set this indicates the trie node is an AnyKey node.
	// Maps to the [AnyKey] component.
	AnyKey *TrieNode

	// Indicates the trie node is an AnyIndex node.
	// Maps to the [AnyIndex] component.
	AnyIndex *TrieNode

	// Set of strings which this trie node matches.
	// Maps to the [Key] component.
	PathKey map[string]*TrieNode

	// Indicates if this node is the end of a pattern. Encountering a node
	// with IsEnd set to true in a trie means the pattern from the root to this
	// node is a complete pattern.
	IsEnd bool
}

// Insert adds a pattern to the trie.
func (t *TrieNode) Insert(pattern Pattern) error {
	// Empty pattern represents the root.
	if len(pattern) == 0 {
		t.IsEnd = true
		return nil
	}

	current := t
	for i, component := range pattern {
		// Create next node based on component type
		var next *TrieNode
		switch c := component.(type) {
		case anyKeyComponent:
			if current.AnyKey == nil {
				current.AnyKey = &TrieNode{}
			}
			next = current.AnyKey

		case anyIndexComponent:
			if current.AnyIndex == nil {
				current.AnyIndex = &TrieNode{}
			}
			next = current.AnyIndex

		case pathComponent:
			if key := c.Key(); key != "" {
				if current.PathKey == nil {
					current.PathKey = make(map[string]*TrieNode)
				}
				if _, exists := current.PathKey[key]; !exists {
					current.PathKey[key] = &TrieNode{}
				}
				next = current.PathKey[key]
			} else {
				return fmt.Errorf("fixed index patterns are not supported: %#v", pattern)
			}
		}

		if next == nil {
			return fmt.Errorf("invalid component type: %T", component)
		}

		// Mark as end of pattern if this is the last component.
		if i == len(pattern)-1 {
			next.IsEnd = true
		}

		// Move to next node
		current = next
	}

	return nil
}

// SearchPath checks if the given path matches any pattern in the trie.
// A path matches if it exactly matches a pattern or if it matches a pattern with wildcards.
func (t *TrieNode) SearchPath(path Path) (Pattern, bool) {
	// We pre-allocate the prefix array that is used to track the current
	// prefix accumulated while walking the prefix tree. Pre-allocating
	// ensures that we do not allocate memory on every recursive call.
	prefix := make(Pattern, len(path))
	pattern, ok := t.searchPathRecursive(t, path, prefix, 0)
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
func (t *TrieNode) searchPathRecursive(node *TrieNode, path Path, prefix Pattern, index int) (Pattern, bool) {
	if node == nil {
		return nil, false
	}

	// Zero case, when the query path is the root node. We return nil here to match
	// the semantics of [MustPatternFromString] which returns nil for the empty string.
	//
	// We cannot return a Pattern{} object here because then MustPatternFromString(""), which
	// returns nil will not be equal to the Pattern{} object returned by this function. An equality
	// is useful because users of this function can use it to check whether the root / empty pattern
	// had been inserted into the trie.
	if len(path) == 0 {
		return nil, node.IsEnd
	}

	// If we've reached the end of the path, check if this node is a valid end of a pattern.
	isLast := index == len(path)
	if isLast {
		return prefix, node.IsEnd
	}

	currentComponent := path[index]

	// First check if the key wildcard is set for the current index.
	if currentComponent.isKey() && node.AnyKey != nil {
		prefix[index] = AnyKey()
		pattern, ok := t.searchPathRecursive(node.AnyKey, path, prefix, index+1)
		if ok {
			return pattern, true
		}
	}

	// If no key wildcard is set, check if the key is an exact match.
	if currentComponent.isKey() {
		child, exists := node.PathKey[currentComponent.Key()]
		if !exists {
			return nil, false
		}
		prefix[index] = currentComponent
		return t.searchPathRecursive(child, path, prefix, index+1)
	}

	if currentComponent.isIndex() && node.AnyIndex != nil {
		prefix[index] = AnyIndex()
		pattern, ok := t.searchPathRecursive(node.AnyIndex, path, prefix, index+1)
		if ok {
			return pattern, true
		}
	}

	// If we've reached this point, the path does not match any patterns in the trie.
	return nil, false
}
