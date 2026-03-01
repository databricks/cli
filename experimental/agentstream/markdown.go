package agentstream

import (
	"fmt"
	"io"
	"regexp"

	"github.com/charmbracelet/glamour"
)

// embeddedBlockRe matches <!-- begin-embedded:... --> ... <!-- end-embedded:... --> blocks.
var embeddedBlockRe = regexp.MustCompile(`(?s)<!-- begin-embedded:\w+ -->\n?(.*?)<!-- end-embedded:\w+ -->`)

// renderMarkdown renders markdown text for the terminal using glamour.
// Strips OneChat embedded query comment blocks before rendering.
func renderMarkdown(w io.Writer, text string) {
	// Strip <!-- begin-embedded:query_xxx --> ... <!-- end-embedded:query_xxx --> wrappers,
	// keeping the content inside (which is a markdown table).
	cleaned := embeddedBlockRe.ReplaceAllString(text, "$1")

	rendered, err := glamour.Render(cleaned, "auto")
	if err != nil {
		// Fall back to plain text on render failure.
		fmt.Fprintln(w, text)
		return
	}
	fmt.Fprint(w, rendered)
}
