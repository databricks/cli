package agentstream

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// embeddedBlockRe matches <!-- begin-embedded:... --> ... <!-- end-embedded:... --> blocks.
var embeddedBlockRe = regexp.MustCompile(`(?s)<!-- begin-embedded:\w+ -->\n?(.*?)<!-- end-embedded:\w+ -->`)

// vizEmbedBlockRe matches embedded viz blocks that are rendered as terminal charts.
// These blocks are removed entirely since the chart is rendered separately via EventViz.
var vizEmbedBlockRe = regexp.MustCompile(`(?s)<!-- begin-embedded:viz_\w+ -->.*?<!-- end-embedded:viz_\w+ -->\n?`)

// vizImageRe matches standalone ![title](#viz_xxx) image references.
var vizImageRe = regexp.MustCompile(`!\[[^\]]*\]\(#viz_\w+\)\n?`)

// cleanMarkdown strips Genie embedded blocks and viz image references.
func cleanMarkdown(text string) string {
	// Remove viz embedded blocks entirely (rendered as terminal charts).
	cleaned := vizEmbedBlockRe.ReplaceAllString(text, "")

	// Remove any remaining standalone viz image references.
	cleaned = vizImageRe.ReplaceAllString(cleaned, "")

	// Strip query embedded block wrappers, keeping the content inside.
	cleaned = embeddedBlockRe.ReplaceAllString(cleaned, "$1")

	// Removed viz blocks can leave a pile of trailing newlines behind;
	// Fprintln supplies the one that should remain.
	return strings.TrimRight(cleaned, "\n")
}

// renderMarkdown prints cleaned markdown as plain text and returns what was
// printed. A message can consist solely of viz references and produce no
// visible output; callers must not count that as an answer.
func renderMarkdown(w io.Writer, text string) string {
	cleaned := cleanMarkdown(text)
	if strings.TrimSpace(cleaned) == "" {
		return ""
	}
	fmt.Fprintln(w, cleaned)
	return cleaned
}
