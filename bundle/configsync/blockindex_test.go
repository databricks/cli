package configsync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadSplitBundle loads a bundle the way the sync command does, so a target's
// overrides are merged into the resources tree and keyed sequences are merged by
// key. Merging keyed sequences happens in the initialize phase, after the target
// is selected, so it has to be applied explicitly here.
func loadSplitBundle(t *testing.T, target, content string) (*bundle.Bundle, *blockResolver) {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks.yml"), []byte(content), 0o600))

	ctx := logdiag.InitContext(t.Context())
	b, err := bundle.Load(ctx, dir)
	require.NoError(t, err)
	mutator.DefaultMutators(ctx, b)
	bundle.ApplyContext(ctx, b, mutator.SelectTarget(target))
	bundle.ApplySeqContext(ctx, b, resourcemutator.MergeJobTasks())

	blocks := newBlockResolver(ctx, b)
	require.NotNil(t, blocks)
	return b, blocks
}

// A task defined in both blocks, contributing a map to it from each.
//
// This case cannot be reached from an acceptance test: structdiff decomposes an object
// change into leaf paths, so a change never addresses a map as a whole, and breaking
// the map branch of winningBlock leaves every fixture passing. It still has to be
// right, because a map's locations do not order the way a scalar's do -- mergeMap
// records the base first even when the target contributed the winning keys.
const twoBlockTaskBundle = `bundle:
  name: split

resources:
  jobs:
    j:
      tasks:
        - task_key: shared
          email_notifications:
            on_failure:
              - base@example.com

targets:
  dev:
    resources:
      jobs:
        j:
          tasks:
            - task_key: shared
              email_notifications:
                no_alert_for_skipped_runs: true
`

func TestBlockResolverRoutesTwoBlockMapToTarget(t *testing.T) {
	b, blocks := loadSplitBundle(t, "dev", twoBlockTaskBundle)

	resolved, err := resolveSelectors("resources.jobs.j.tasks[task_key='shared'].email_notifications", b, OperationReplace)
	require.NoError(t, err)

	destination, err := blocks.singleDestination(resolved)
	require.NoError(t, err)
	assert.True(t, destination.block.override)
	assert.Equal(t, "resources.jobs.j.tasks[0].email_notifications", destination.path.String())
}
