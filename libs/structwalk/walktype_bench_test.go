package structwalk

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structdiff/structpath"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func countFields(typ reflect.Type) (int, error) {
	fieldCount := 0
	err := WalkType(typ, func(path *structpath.PathNode, typ reflect.Type) (continueWalk bool) {
		fieldCount++
		return true
	})
	return fieldCount, err
}

func benchmarkWalkType(b *testing.B, tt reflect.Type) {
	total := 0

	b.ResetTimer()
	for range b.N {
		count, err := countFields(tt)
		if err != nil {
			b.Fatalf("WalkType failed: %v", err)
		}
		total += count
	}

	b.StopTimer()
	// Root now correctly includes embedded struct fields, so it has many more fields than JobSettings
	// (3,487 vs 533) because it contains JobSettings plus other resource types and config fields
	b.Logf("Counted fields in %s: %d (%d/%d)", tt, total/b.N, total, b.N)
}

func BenchmarkWalkTypeJobSettings(b *testing.B) {
	benchmarkWalkType(b, reflect.TypeOf(jobs.JobSettings{}))
}

func BenchmarkWalkTypeRoot(b *testing.B) {
	benchmarkWalkType(b, reflect.TypeOf(config.Root{}))
}
