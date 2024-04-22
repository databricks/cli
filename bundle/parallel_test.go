package bundle

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/require"
)

type addToContainer struct {
	container *[]int
	value     int
	err       bool
}

func (m *addToContainer) Apply(ctx context.Context, b ReadOnlyBundle) diag.Diagnostics {
	if m.err {
		return diag.Errorf("error")
	}

	c := *m.container
	c = append(c, m.value)
	*m.container = c
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
	m1 := &addToContainer{container: &container, value: 1}
	m2 := &addToContainer{container: &container, value: 2}
	m3 := &addToContainer{container: &container, value: 3}

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
	m1 := &addToContainer{container: &container, value: 1}
	m2 := &addToContainer{container: &container, err: true, value: 2}
	m3 := &addToContainer{container: &container, value: 3}

	m := Parallel(m1, m2, m3)

	// Apply the mutator
	diags := ApplyReadOnly(context.Background(), ReadOnly(b), m)
	require.Len(t, diags, 1)
	require.Equal(t, "error", diags[0].Summary)
	require.Len(t, container, 2)
	require.Contains(t, container, 1)
	require.Contains(t, container, 3)
}
