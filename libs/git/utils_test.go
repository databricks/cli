package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToHttpsUrlForSsh(t *testing.T) {
	for _, e := range []struct {
		url      string
		expected string
	}{
		{"user@foo.com:org/repo-name.git", "https://foo.com/org/repo-name"},
		{"git@github.com:databricks/cli.git", "https://github.com/databricks/cli"},
		{"https://github.com/databricks/cli.git", "https://github.com/databricks/cli"},
	} {
		url, err := ToHttpsUrl(e.url)
		assert.NoError(t, err)
		assert.Equal(t, e.expected, url)
	}
}
