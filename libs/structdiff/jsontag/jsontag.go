package jsontag

import "strings"

// JSONTag represents a struct field's `json` tag as a string.
// It provides methods to lazily extract information from the tag.
type JSONTag string

// Name returns the field name that JSON should use.
// Returns "" if no name is specified (meaning "keep Go field name"),
// or "-" if the field should be skipped.
func (tag JSONTag) Name() string {
	s := string(tag)
	if s == "" {
		return ""
	}

	if idx := strings.IndexByte(s, ','); idx == -1 {
		// Whole tag is just the name
		return s
	} else {
		return s[:idx]
	}
}

// OmitEmpty returns true if the tag contains "omitempty" option.
func (tag JSONTag) OmitEmpty() bool {
	return tag.hasOption("omitempty")
}

// OmitZero returns true if the tag contains "omitzero" option.
func (tag JSONTag) OmitZero() bool {
	return tag.hasOption("omitzero")
}

// hasOption checks if the tag contains the specified option.
func (tag JSONTag) hasOption(option string) bool {
	s := string(tag)
	if s == "" {
		return false
	}

	// Skip the name part
	if idx := strings.IndexByte(s, ','); idx == -1 {
		// No options, just name
		return false
	} else {
		s = s[idx+1:]
	}

	// Walk the comma-separated options
	for len(s) > 0 {
		opt := s
		if i := strings.IndexByte(s, ','); i != -1 {
			opt, s = s[:i], s[i+1:]
		} else {
			s = ""
		}

		if opt == option {
			return true
		}
	}
	return false
}
