package sync

import (
	"context"
	"encoding/json"
	"io"
	"log"

	"github.com/databricks/bricks/libs/sync"
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

// Read synchronization events and log them at the INFO level.
func logOutput(ctx context.Context, ch <-chan sync.Event) {
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
				log.Printf("[INFO] %s", e.String())
			}
		}
	}
}
