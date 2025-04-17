package dyn

import (
	"fmt"
	"strconv"
	"strings"
)

// MustPathFromString is like NewPathFromString but panics on error.
func MustPathFromString(input string) Path {
	p, err := NewPathFromString(input)
	if err != nil {
		panic(err)
	}
	return p
}

// NewPathFromString parses a path from a string.
//
// The string must be a sequence of keys and indices separated by dots.
// Indices must be enclosed in square brackets.
// The string may include a leading dot.
//
// Examples:
//   - foo.bar
//   - foo[1].bar
//   - foo.bar[1]
//   - foo.bar[1][2]
//   - .
func NewPathFromString(input string) (Path, error) {
	var path Path

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
			return nil, fmt.Errorf("invalid path: %s", input)
		}

		if p[0] == '[' {
			// Find next ]
			i := strings.Index(p, "]")
			if i < 0 {
				return nil, fmt.Errorf("invalid path: %s", input)
			}

			// Parse index
			j, err := strconv.Atoi(p[1:i])
			if err != nil {
				return nil, fmt.Errorf("invalid path: %s", input)
			}

			// Append index
			path = append(path, Index(j))
			p = p[i+1:]

			// The next character must be a . or [
			if p != "" && strings.IndexAny(p, ".[") != 0 {
				return nil, fmt.Errorf("invalid path: %s", input)
			}
		} else {
			// Find next . or [
			i := strings.IndexAny(p, ".[")
			if i < 0 {
				i = len(p)
			}

			if i == 0 {
				return nil, fmt.Errorf("invalid path: %s", input)
			}

			// Append key
			path = append(path, Key(p[:i]))
			p = p[i:]
		}
	}

	return path, nil
}
