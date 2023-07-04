package sync

import (
	"bufio"
	"context"
	"encoding/json"
	"io"

	"github.com/databricks/cli/libs/sync"
)

// Read synchronization events and write them as JSON to the specified writer (typically stdout).
func jsonOutput(ctx context.Context, ch <-chan sync.Event, w io.Writer) {
	enc := json.NewEncoder(w)
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-ch:
			if !ok {
				return
			}
			err := enc.Encode(e)
			// These are plain structs so this must always work.
			// Panic on error so that a violation of this assumption does not go undetected.
			if err != nil {
				panic(err)
			}
		}
	}
}

// Read synchronization events and write them as text to the specified writer (typically stdout).
func textOutput(ctx context.Context, ch <-chan sync.Event, w io.Writer) {
	bw := bufio.NewWriter(w)

	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-ch:
			if !ok {
				return
			}
			// Log only if something actually happened.
			// Sync events produce an empty string if nothing happened.
			if str := e.String(); str != "" {
				bw.WriteString(str)
				bw.WriteString("\n")
				bw.Flush()
			}
		}
	}
}
