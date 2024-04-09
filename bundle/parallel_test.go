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
	err       bool
}

func (m *addToContainer) Apply(ctx context.Context, b *Bundle) diag.Diagnostics {
	if m.err {
		return diag.Errorf("error")
	}

	c := *m.container
	c = append(c, 1)
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
	m1 := &addToContainer{container: &container}
	m2 := &addToContainer{container: &container}
	m3 := &addToContainer{container: &container}

	m := Parallel(m1, m2, m3)

	// Apply the mutator
	diags := m.Apply(context.Background(), b)
	require.Empty(t, diags)
	require.Len(t, container, 3)
}

func TestParallelMutatorWorkWithErrors(t *testing.T) {
	b := &Bundle{
		Config: config.Root{},
	}

	container := []int{}
	m1 := &addToContainer{container: &container}
	m2 := &addToContainer{container: &container, err: true}
	m3 := &addToContainer{container: &container}

	m := Parallel(m1, m2, m3)

	// Apply the mutator
	diags := m.Apply(context.Background(), b)
	require.Len(t, diags, 1)
	require.Equal(t, "error", diags[0].Summary)
	require.Len(t, container, 2)
}
