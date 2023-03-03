package sync

import (
	"context"
	"encoding/json"
	"io"
	"log"

	"github.com/databricks/bricks/libs/sync"
)

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
			enc.Encode(e)
		}
	}
}

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
