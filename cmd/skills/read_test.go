package skills

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShellQuote(t *testing.T) {
	assert.Equal(t, "references/guide.md", shellQuote("references/guide.md"))
	assert.Equal(t, "databricks-bundles", shellQuote("databricks-bundles"))
	assert.Equal(t, "''", shellQuote(""))
	assert.Equal(t, "'references/my guide.md'", shellQuote("references/my guide.md"))
	assert.Equal(t, "'$(rm -rf ~)'", shellQuote("$(rm -rf ~)"))
	assert.Equal(t, `'it'\''s'`, shellQuote("it's"))
}

func TestWriteReadTextQuotesFileCommands(t *testing.T) {
	var out bytes.Buffer
	err := writeReadText(&out, "guide-skill", []byte("# Guide\n"), []string{"references/my guide.md"})
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "  databricks skills read-file guide-skill 'references/my guide.md'\n")
}
