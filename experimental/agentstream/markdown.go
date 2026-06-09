package agentstream

import (
	"fmt"
	"io"
	"regexp"

	"github.com/charmbracelet/glamour"
)

// embeddedBlockRe matches <!-- begin-embedded:... --> ... <!-- end-embedded:... --> blocks.
var embeddedBlockRe = regexp.MustCompile(`(?s)<!-- begin-embedded:\w+ -->\n?(.*?)<!-- end-embedded:\w+ -->`)

// vizEmbedBlockRe matches embedded viz blocks that are rendered as terminal charts.
// These blocks are removed entirely since the chart is rendered separately via EventViz.
var vizEmbedBlockRe = regexp.MustCompile(`(?s)<!-- begin-embedded:viz_\w+ -->.*?<!-- end-embedded:viz_\w+ -->\n?`)

// vizImageRe matches standalone ![title](#viz_xxx) image references.
var vizImageRe = regexp.MustCompile(`!\[[^\]]*\]\(#viz_\w+\)\n?`)

// renderMarkdown renders markdown text for the terminal using glamour.
// Strips Genie embedded blocks and viz image references before rendering.
func renderMarkdown(w io.Writer, text string) {
	// Remove viz embedded blocks entirely (rendered as terminal charts).
	cleaned := vizEmbedBlockRe.ReplaceAllString(text, "")

	// Remove any remaining standalone viz image references.
	cleaned = vizImageRe.ReplaceAllString(cleaned, "")

	// Strip query embedded block wrappers, keeping the content inside.
	cleaned = embeddedBlockRe.ReplaceAllString(cleaned, "$1")

	rendered, err := glamour.Render(cleaned, "auto")
	if err != nil {
		fmt.Fprintln(w, text)
		return
	}
	fmt.Fprint(w, rendered)
}
