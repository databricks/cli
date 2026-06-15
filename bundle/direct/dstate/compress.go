package dstate

import (
	"bytes"
	"compress/flate"
)

// compressedStateSize returns the DEFLATE-compressed size in bytes of a
// resource's serialized state. It is a rough proxy, used purely for deploy
// telemetry, for what the state sizes look like on the server side (which
// compresses with zstd): we deliberately use the standard library's
// compress/flate rather than pull in a dedicated zstd dependency, keeping the
// supply chain small while still getting useful signal on compressibility.
// Returns 0 for empty state.
//
// This always terminates: DEFLATE is a single linear pass over a finite,
// in-memory buffer (no input loop can diverge — non-termination is a
// decompression concern, e.g. zip bombs, not compression), and the writer
// targets a bytes.Buffer that never blocks. Cost is O(len(state)) — a few
// milliseconds even for the largest (~1 MB) resource states — so a background
// goroutine is not warranted.
func compressedStateSize(state []byte) int {
	if len(state) == 0 {
		return 0
	}
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		return 0
	}
	if _, err := w.Write(state); err != nil {
		return 0
	}
	if err := w.Close(); err != nil {
		return 0
	}
	return buf.Len()
}

// compressStateSizes returns each resource's compressed state size, keyed by
// resource key. Compression is independent per resource and dominates the cost,
// so each resource is compressed in its own goroutine (the runtime schedules
// them across cores; no worker pool needed) and the result is sent on a
// buffered channel that the caller drains into the map.
//
// This always terminates: the channel is buffered to the resource count, so
// every goroutine sends exactly once and exits without blocking, and the drain
// loop reads exactly that many results — no goroutine leaks, no deadlock.
func compressStateSizes(data Database) map[string]int {
	type result struct {
		key  string
		size int
	}

	ch := make(chan result, len(data.State))
	for key, entry := range data.State {
		go func() {
			ch <- result{key: key, size: compressedStateSize(entry.State)}
		}()
	}

	sizes := make(map[string]int, len(data.State))
	for range data.State {
		r := <-ch
		sizes[r.key] = r.size
	}
	return sizes
}
