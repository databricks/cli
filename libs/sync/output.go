package sync

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
)

// Read synchronization events and write them as JSON to the specified writer (typically stdout).
func JsonOutput(ctx context.Context, ch <-chan Event, w io.Writer) {
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
func TextOutput(ctx context.Context, ch <-chan Event, w io.Writer) {
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
				_, _ = bw.WriteString(str)
				_, _ = bw.WriteString("\n")
				_ = bw.Flush()
			}
		}
	}
}
