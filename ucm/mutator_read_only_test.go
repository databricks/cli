package ucm_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
	"github.com/stretchr/testify/assert"
)

type countingROMutator struct {
	ucm.RO
	name  string
	count *atomic.Int32
}

func (m *countingROMutator) Name() string { return m.name }

func (m *countingROMutator) Apply(_ context.Context, _ *ucm.Ucm) diag.Diagnostics {
	m.count.Add(1)
	return nil
}

func TestApplyParallelRunsAllMutators(t *testing.T) {
	u := &ucm.Ucm{}
	var c1, c2, c3 atomic.Int32

	ucm.ApplyParallel(t.Context(), u,
		&countingROMutator{name: "m1", count: &c1},
		&countingROMutator{name: "m2", count: &c2},
		&countingROMutator{name: "m3", count: &c3},
	)

	assert.Equal(t, int32(1), c1.Load())
	assert.Equal(t, int32(1), c2.Load())
	assert.Equal(t, int32(1), c3.Load())
}

func TestApplyParallelWithNoMutators(t *testing.T) {
	u := &ucm.Ucm{}
	ucm.ApplyParallel(t.Context(), u)
}
