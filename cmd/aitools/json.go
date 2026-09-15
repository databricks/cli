package aitools

import (
	"encoding/json"
	"io"
)

// renderJSON writes v as indented JSON. Shared by the install and list commands
// so their --output json payloads are formatted identically.
func renderJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
