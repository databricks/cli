package structdiff

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func bench(b *testing.B, job1, job2 string) {
	var x, y jobs.JobSettings

	require.NoError(b, json.Unmarshal([]byte(job1), &x))
	require.NoError(b, json.Unmarshal([]byte(job2), &y))

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

func BenchmarkEqual(b *testing.B) {
	bench(b, jobExampleResponse, jobExampleResponse)
}

func BenchmarkChanges(b *testing.B) {
	job2 := strings.ReplaceAll(jobExampleResponse, "1", "2")
	bench(b, jobExampleResponse, job2)
}

func BenchmarkZero(b *testing.B) {
	bench(b, jobExampleResponse, jobExampleResponseZeroes)
}

func BenchmarkNils(b *testing.B) {
	bench(b, jobExampleResponse, jobExampleResponseNils)
}
