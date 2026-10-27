package dyn

import (
	"fmt"
	"strconv"
	"strings"
)

// MustPatternFromString is like NewPatternFromString but panics on error.
func MustPatternFromString(input string) Pattern {
	p, err := NewPatternFromString(input)
	if err != nil {
		panic(err)
	}
	return p
}

// NewPatternFromString parses a pattern from a string.
//
// The string must be a sequence of keys and indices separated by dots.
// Indices must be enclosed in square brackets.
// The string may include a leading dot.
// The wildcard character '*' can be used to match any key or index.
//
// Examples:
//   - foo.bar
//   - foo[1].bar
//   - foo.*.bar
//   - foo[*].bar
//   - .
func NewPatternFromString(input string) (Pattern, error) {
	var pattern Pattern

	p := input

	// Trim leading dot.
	if p != "" && p[0] == '.' {
		p = p[1:]
	}

	for p != "" {
		// Every component may have a leading dot.
		if p[0] == '.' {
			p = p[1:]
		}

		if p == "" {
			return nil, fmt.Errorf("invalid pattern: %s", input)
		}

		if p[0] == '[' {
			// Find next ]
			i := strings.Index(p, "]")
			if i < 0 {
				return nil, fmt.Errorf("invalid pattern: %s", input)
			}

			// Check for wildcard
			if p[1:i] == "*" {
				pattern = append(pattern, AnyIndex())
			} else {
				// Parse index
				j, err := strconv.Atoi(p[1:i])
				if err != nil {
					return nil, fmt.Errorf("invalid pattern: %s", input)
				}

				// Append index
				pattern = append(pattern, Index(j))
			}

			p = p[i+1:]

			// The next character must be a . or [
			if p != "" && strings.IndexAny(p, ".[") != 0 {
				return nil, fmt.Errorf("invalid pattern: %s", input)
			}
		} else {
			// Find next . or [
			i := strings.IndexAny(p, ".[")
			if i < 0 {
				i = len(p)
			}

			if i == 0 {
				return nil, fmt.Errorf("invalid pattern: %s", input)
			}

			// Check for wildcard
			if p[:i] == "*" {
				pattern = append(pattern, AnyKey())
			} else {
				// Append key
				pattern = append(pattern, Key(p[:i]))
			}

			p = p[i:]
		}
	}

	return pattern, nil
}
