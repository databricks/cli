package ai

import (
	"context"
	"time"

	"github.com/databricks/cli/libs/cmdio"
)

// envelopeVersion is the envelope's format-version marker. The Python `air` CLI
// hardcodes it to 1; it lets consumers detect a future incompatible change to
// the envelope shape.
const envelopeVersion = 1

// envelope is the JSON shape that the AI runtime CLI prints:
//
//	{ "v": 1, "ts": "2024-01-15T14:30:45Z", "data": { ... } }
//
// It mirrors the envelope used by the original Python `air` CLI so existing
// consumers keep working after the port to Go.
type envelope struct {
	// V is the envelope format-version marker (always 1).
	V int `json:"v"`
	// TS is the wall-clock time the response was produced, in RFC 3339 UTC.
	// It is an absolute timestamp, not an elapsed duration.
	TS string `json:"ts"`
	// Data is the command-specific payload.
	Data any `json:"data"`
}

// renderEnvelope wraps data in the JSON envelope and prints it.
// Fields that should appear only in text output are tagged `json:"-"` on the payload struct.
func renderEnvelope(ctx context.Context, data any) error {
	return cmdio.Render(ctx, envelope{
		V:    envelopeVersion,
		TS:   time.Now().UTC().Format(time.RFC3339),
		Data: data,
	})
}
