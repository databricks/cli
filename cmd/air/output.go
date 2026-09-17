package aircmd

import (
	"context"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
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

// jsonError is the error payload, matching the Python `air` CLI's shape (cli/json_output.py).
type jsonError struct {
	Code      string `json:"code"`
	Kind      string `json:"kind"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

// errorEnvelope is what a failed command prints in JSON mode:
//
//	{ "v": 1, "ts": "...", "error": { "code": ..., "kind": ..., "message": ..., "retryable": ... } }
type errorEnvelope struct {
	V     int       `json:"v"`
	TS    string    `json:"ts"`
	Error jsonError `json:"error"`
}

// renderError prints err as a JSON error envelope when output is JSON, returning
// root.ErrAlreadyPrinted so the command exits non-zero without Cobra reprinting
// it; in text mode it returns err unchanged. code/kind/retryable match the
// Python CLI's call site.
func renderError(ctx context.Context, cmd *cobra.Command, code, kind string, retryable bool, err error) error {
	if root.OutputType(cmd) != flags.OutputJSON {
		return err
	}
	if rerr := cmdio.Render(ctx, errorEnvelope{
		V:  envelopeVersion,
		TS: time.Now().UTC().Format(time.RFC3339),
		Error: jsonError{
			Code:      code,
			Kind:      kind,
			Message:   err.Error(),
			Retryable: retryable,
		},
	}); rerr != nil {
		return rerr
	}
	return root.ErrAlreadyPrinted
}
