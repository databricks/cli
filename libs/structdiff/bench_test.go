package structdiff

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func BenchmarkEqualJobSettings(b *testing.B) {
	var x, y jobs.JobSettings

	require.NoError(b, json.Unmarshal([]byte(jobExampleResponse), &x))
	require.NoError(b, json.Unmarshal([]byte(jobExampleResponse), &y))

	total := 0

	b.ResetTimer()
	for range b.N {
		changes, err := GetStructDiff(&x, &y)
		if err != nil {
			b.Fatalf("error: %s", err)
		}
		total += len(changes)
	}
	b.StopTimer()

	b.Logf("Total: %d / %d", total, b.N)
}

func BenchmarkDiffJobSettings(b *testing.B) {
	var x, y jobs.JobSettings

	require.NoError(b, json.Unmarshal([]byte(jobExampleResponse), &x))

	resp2 := strings.ReplaceAll(jobExampleResponse, "1", "2")
	require.NoError(b, json.Unmarshal([]byte(resp2), &y))

	total := 0

	b.ResetTimer()
	for range b.N {
		changes, err := GetStructDiff(&x, &y)
		if err != nil {
			b.Fatalf("error: %s", err)
		}
		total += len(changes)
	}
	b.StopTimer()

	b.Logf("Total: %d / %d", total, b.N)
}
