package bundle

import (
	"context"
	"sync"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/require"
)

type addToContainer struct {
	t         *testing.T
	container *[]int
	value     int
	err       bool

	// mu is a mutex that protects container. It is used to ensure that the
	// container slice is only modified by one goroutine at a time.
	mu *sync.Mutex
}

func (m *addToContainer) Apply(ctx context.Context, b ReadOnlyBundle) diag.Diagnostics {
	if m.err {
		return diag.Errorf("error")
	}

	m.mu.Lock()
	*m.container = append(*m.container, m.value)
	m.mu.Unlock()

	return nil
}

func (m *addToContainer) Name() string {
	return "addToContainer"
}

func TestParallelMutatorWork(t *testing.T) {
	b := &Bundle{
		Config: config.Root{},
	}

	container := []int{}
	var mu sync.Mutex
	m1 := &addToContainer{t: t, container: &container, value: 1, mu: &mu}
	m2 := &addToContainer{t: t, container: &container, value: 2, mu: &mu}
	m3 := &addToContainer{t: t, container: &container, value: 3, mu: &mu}

	m := Parallel(m1, m2, m3)

	// Apply the mutator
	diags := ApplyReadOnly(context.Background(), ReadOnly(b), m)
	require.Empty(t, diags)
	require.Len(t, container, 3)
	require.Contains(t, container, 1)
	require.Contains(t, container, 2)
	require.Contains(t, container, 3)
}

func TestParallelMutatorWorkWithErrors(t *testing.T) {
	b := &Bundle{
		Config: config.Root{},
	}

	container := []int{}
	var mu sync.Mutex
	m1 := &addToContainer{container: &container, value: 1, mu: &mu}
	m2 := &addToContainer{container: &container, err: true, value: 2, mu: &mu}
	m3 := &addToContainer{container: &container, value: 3, mu: &mu}

	m := Parallel(m1, m2, m3)

	// Apply the mutator
	diags := ApplyReadOnly(context.Background(), ReadOnly(b), m)
	require.Len(t, diags, 1)
	require.Equal(t, "error", diags[0].Summary)
	require.Len(t, container, 2)
	require.Contains(t, container, 1)
	require.Contains(t, container, 3)
}
