package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNextLink(t *testing.T) {
	tests := []struct {
		name       string
		linkHeader string
		expected   string
	}{
		// First and foremost, the well-formed cases that we can expect from real GitHub API.
		{
			name:       "no header",
			linkHeader: "",
			expected:   "",
		},
		{
			name:       "documentation example",
			linkHeader: `<https://api.github.com/repositories/1300192/issues?page=2>; rel="prev", <https://api.github.com/repositories/1300192/issues?page=4>; rel="next", <https://api.github.com/repositories/1300192/issues?page=515>; rel="last", <https://api.github.com/repositories/1300192/issues?page=1>; rel="first"`,
			expected:   "https://api.github.com/repositories/1300192/issues?page=4",
		},
		{
			name:       "with next only",
			linkHeader: `<https://api.github.com/repos/databricks/cli/issues?page=2>; rel="next"`,
			expected:   "https://api.github.com/repos/databricks/cli/issues?page=2",
		},
		{
			name:       "without next",
			linkHeader: `<https://api.github.com/repositories/1300192/issues?page=1>; rel="prev", <https://api.github.com/repositories/1300192/issues?page=1>; rel="first", <https://api.github.com/repositories/1300192/issues?page=515>; rel="last"`,
			expected:   "",
		},
		{
			name:       "next at beginning",
			linkHeader: `<https://api.github.com/repos/test/test?page=5>; rel="next", <https://api.github.com/repos/test/test?page=10>; rel="last"`,
			expected:   "https://api.github.com/repos/test/test?page=5",
		},
		{
			name:       "next at end",
			linkHeader: `<https://api.github.com/repos/test/test?page=10>; rel="last", <https://api.github.com/repos/test/test?page=5>; rel="next"`,
			expected:   "https://api.github.com/repos/test/test?page=5",
		},
		// Malformed cases to ensure robustness. (These should not occur in practice, but are here to demonstrate resilience.)
		{
			name:       "malformed no semicolon",
			linkHeader: `<https://api.github.com/repos/test/test?page=2> rel="next"`,
			expected:   "",
		},
		{
			name:       "malformed no angle-brackets",
			linkHeader: `https://api.github.com/repos/test/test?page=2; rel="next"`,
			expected:   "",
		},
		{
			name:       "malformed multiple parts",
			linkHeader: `<https://api.github.com/repos/test/test?page=2>; rel="next"; extra="value"`,
			expected:   "",
		},
		{
			name:       "malformed no url",
			linkHeader: `<>; rel="next"`,
			expected:   "",
		},
		{
			name:       "malformed empty link",
			linkHeader: `, <https://api.github.com/repos/test/test?page=5>; rel="next"`,
			expected:   "https://api.github.com/repos/test/test?page=5",
		},
		// Borderline case: some tolerance of whitespace.
		{
			name:       "tolerate whitespace",
			linkHeader: `  <https://api.github.com/repos/test/test?page=2>  ;  rel="next"  `,
			expected:   "https://api.github.com/repos/test/test?page=2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNextLink(tt.linkHeader)
			assert.Equal(t, tt.expected, result)
		})
	}
}
