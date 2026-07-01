package dstate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompressedStateSize(t *testing.T) {
	// Empty state has no compressed size.
	assert.Equal(t, 0, compressedStateSize(nil))
	assert.Equal(t, 0, compressedStateSize([]byte{}))

	// A highly compressible blob shrinks: positive size, smaller than raw.
	blob := bytes.Repeat([]byte(`{"key":"value"}`), 1000)
	got := compressedStateSize(blob)
	assert.Positive(t, got)
	assert.Less(t, got, len(blob))
}

// jsonState builds a JSON-ish resource-state blob of about `size` bytes:
// repeated keys with varied values, representative of real resource state
// (compressible, but not trivially so). `seed` varies the content so different
// resources don't compress identically. The exact byte count is approximate.
func jsonState(size, seed int) json.RawMessage {
	r := rand.New(rand.NewPCG(uint64(seed), 0))
	b := make([]byte, 0, size+128)
	b = append(b, '{')
	for i := 0; len(b) < size; i++ {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, fmt.Sprintf(
			`"key_%d":{"node_type":"i3.xlarge","num_workers":%d,"path":"/Workspace/Users/u%d/resource_%d","tag":"%x"}`,
			i, r.IntN(64), r.IntN(100000), i, r.Uint64(),
		)...)
	}
	b = append(b, '}')
	return json.RawMessage(b)
}

// Representative results (go1.26.4, linux/amd64, Intel Xeon Platinum 8375C @
// 2.90GHz; realistic varied-JSON state — absolute numbers are hardware- and
// data-dependent):
//
//	BenchmarkCompressedStateSize/4KB        211 µs     19 MB/s
//	BenchmarkCompressedStateSize/64KB       1.3 ms     49 MB/s
//	BenchmarkCompressedStateSize/256KB      4.9 ms     54 MB/s
//	BenchmarkCompressedStateSize/1024KB    19.0 ms     55 MB/s   (largest per-resource state ~1 MB)
//	BenchmarkExportStateFromData/200x8KB   22.2 ms     74 MB/s
//	BenchmarkExportStateFromData/50x64KB    9.5 ms    345 MB/s
//	BenchmarkExportStateFromData/8x1MB     26.0 ms    323 MB/s   (~6x vs sequential via per-resource fan-out)
//	BenchmarkExportStateFromData/1x1MB     19.4 ms     54 MB/s   (single blob)

// BenchmarkCompressedStateSize measures per-resource compression cost. The
// largest per-resource state files are around 1 MB, so 1024 KB is the top size.
func BenchmarkCompressedStateSize(b *testing.B) {
	for _, kb := range []int{4, 64, 256, 1024} {
		data := jsonState(kb<<10, 1)
		b.Run(fmt.Sprintf("%dKB", kb), func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for b.Loop() {
				_ = compressedStateSize(data)
			}
		})
	}
}

// BenchmarkExportStateFromData measures the full export (including the
// per-resource compression fan-out) for a few representative bundle shapes,
// from many small resources to a few ~1 MB ones.
func BenchmarkExportStateFromData(b *testing.B) {
	cases := []struct {
		name        string
		count, size int
	}{
		{"200x8KB", 200, 8 << 10},
		{"50x64KB", 50, 64 << 10},
		{"8x1MB", 8, 1 << 20},
		{"1x1MB", 1, 1 << 20},
	}
	for _, c := range cases {
		data := Database{State: make(map[string]ResourceEntry, c.count)}
		var total int64
		for i := range c.count {
			st := jsonState(c.size, i)
			total += int64(len(st))
			data.State[fmt.Sprintf("resources.jobs.job_%d", i)] = ResourceEntry{
				ID:    strconv.Itoa(i),
				State: st,
			}
		}
		b.Run(c.name, func(b *testing.B) {
			b.SetBytes(total)
			for b.Loop() {
				_ = ExportStateFromData(data)
			}
		})
	}
}
