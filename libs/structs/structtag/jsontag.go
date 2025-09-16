package structtag

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

func (tag JSONTag) OmitEmpty() bool {
	return tag.hasOption("omitempty")
}

func (tag JSONTag) OmitZero() bool {
	return tag.hasOption("omitzero")
}

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

	return hasOption(s, option)
}

func hasOption(tag, option string) bool {
	// Walk the comma-separated options
	for len(tag) > 0 {
		opt := tag
		if i := strings.IndexByte(tag, ','); i != -1 {
			opt, tag = tag[:i], tag[i+1:]
		} else {
			tag = ""
		}

		if opt == option {
			return true
		}
	}
	return false
}
