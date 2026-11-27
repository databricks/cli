package cmdio

import (
	"io"
	"strings"
	"text/template"

	"github.com/databricks/cli/libs/textutil"
)

// executeTemplate executes t on data and writes the result to w with terminal
// control characters escaped. What a template prints mostly comes from an API
// response, and text/template (unlike the JSON encoder) writes strings as they
// are. The rendered text is escaped rather than data, so that everything the
// template prints is covered: fields, method results, template functions.
// keepColors says whether the template or the values the CLI styled for it can
// have colors in them; see escapeComposed.
func executeTemplate(t *template.Template, w io.Writer, data any, keepColors bool) error {
	var rendered strings.Builder
	err := t.Execute(&rendered, data)
	text := rendered.String()
	if keepColors {
		// Most templates render table rows: a style a value leaves on ends
		// with its cell.
		text = textutil.EscapeControlKeepColorsPerCell(text)
	} else {
		text = textutil.EscapeControl(text)
	}
	// As when executing on w itself, what was rendered before an error is written.
	if _, writeErr := io.WriteString(w, text); err == nil {
		err = writeErr
	}
	return err
}

// escapeComposed escapes terminal control characters in text the CLI composed
// from its own text and other values, such as a name from an API response. With
// colors on, the CLI's own colors are part of the text and cannot be told from
// a color in a value, so colors are kept. With colors off the CLI adds none, so
// nothing is kept.
func escapeComposed(s string, keepColors bool) string {
	if keepColors {
		return textutil.EscapeControlKeepColors(s)
	}
	return textutil.EscapeControl(s)
}

// colorsInLogs reports whether text written to the error writer can have the
// CLI's own colors in it. The color helpers follow stdout and renderers for the
// error writer follow stderr, so it is either.
func (c *cmdIO) colorsInLogs() bool {
	return c.capabilities.SupportsStdoutColor() || c.capabilities.SupportsStderrColor()
}
