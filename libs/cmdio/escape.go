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
// Colors are kept, because the template's own colors are in that text too.
func executeTemplate(t *template.Template, w io.Writer, data any) error {
	var rendered strings.Builder
	err := t.Execute(&rendered, data)
	// As when executing on w itself, what was rendered before an error is written.
	if _, writeErr := io.WriteString(w, textutil.EscapeControlKeepColors(rendered.String())); err == nil {
		err = writeErr
	}
	return err
}
