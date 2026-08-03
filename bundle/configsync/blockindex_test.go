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

// A task defined in both blocks, where one scalar and one map are contributed by
// each. Merging records a scalar's winning location first but a map's *base*
// location first, so the two kinds disagree about what Locations()[0] means.
const twoBlockTaskBundle = `bundle:
  name: split

resources:
  jobs:
    j:
      tasks:
        - task_key: shared
          max_retries: 1
          email_notifications:
            on_failure:
              - base@example.com
        - task_key: top_only
          max_retries: 5

targets:
  dev:
    resources:
      jobs:
        j:
          tasks:
            - task_key: shared
              max_retries: 2
              email_notifications:
                no_alert_for_skipped_runs: true
            - task_key: target_only
              max_retries: 6
`

func TestBlockResolverRoutesToDefiningBlock(t *testing.T) {
	b, blocks := loadSplitBundle(t, "dev", twoBlockTaskBundle)

	tests := []struct {
		name     string
		path     string
		override bool
		want     string
	}{
		{
			name:     "field defined only top-level",
			path:     "resources.jobs.j.tasks[task_key='top_only'].max_retries",
			override: false,
			want:     "resources.jobs.j.tasks[1].max_retries",
		},
		{
			name:     "field defined only in the target",
			path:     "resources.jobs.j.tasks[task_key='target_only'].max_retries",
			override: true,
			want:     "resources.jobs.j.tasks[1].max_retries",
		},
		{
			// A scalar in both blocks: the target's value is the deployed one, so
			// writing the top-level copy would not change the effective value.
			name:     "scalar defined in both blocks routes to the target",
			path:     "resources.jobs.j.tasks[task_key='shared'].max_retries",
			override: true,
			want:     "resources.jobs.j.tasks[0].max_retries",
		},
		{
			// A leaf under the map still has exactly one definition of its own.
			name:     "leaf under a two-block map routes to its own block",
			path:     "resources.jobs.j.tasks[task_key='shared'].email_notifications.on_failure",
			override: false,
			want:     "resources.jobs.j.tasks[0].email_notifications.on_failure",
		},
		{
			// The map as a whole: mergeMap records the base's location first even
			// though the target contributed keys, so the first location says nothing
			// about precedence and the narrower scope is used instead.
			name:     "map defined in both blocks routes to the target",
			path:     "resources.jobs.j.tasks[task_key='shared'].email_notifications",
			override: true,
			want:     "resources.jobs.j.tasks[0].email_notifications",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolved, err := resolveSelectors(tc.path, b, OperationReplace)
			require.NoError(t, err)

			destination, err := blocks.singleDestination(resolved)
			require.NoError(t, err)
			assert.Equal(t, tc.override, destination.block.override)
			assert.Equal(t, tc.want, destination.path.String())
		})
	}
}

func TestBlockResolverRoutesElementToEveryDefiningBlock(t *testing.T) {
	b, blocks := loadSplitBundle(t, "dev", twoBlockTaskBundle)

	// Removing or renaming an element defined in both blocks has to reach the part
	// in each, and each part has its own index inside its own block.
	resolved, err := resolveSelectors("resources.jobs.j.tasks[task_key='shared']", b, OperationRemove)
	require.NoError(t, err)

	destinations, err := blocks.routeDestinations(resolved)
	require.NoError(t, err)

	got := map[bool]string{}
	for _, d := range destinations {
		got[d.block.override] = d.path.String()
	}
	assert.Equal(t, map[bool]string{
		false: "resources.jobs.j.tasks[0]",
		true:  "resources.jobs.j.tasks[0]",
	}, got)
}

func TestBlockResolverCountsIndexPerFile(t *testing.T) {
	// One target's override spread over two included files. Resource keys are
	// unique across top-level files but that is not enforced inside targets, so a
	// task second in the concatenated target region can be first in the file that
	// defines it -- and the write has to use the latter.
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "overrides"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks.yml"), []byte(`bundle:
  name: split

include:
  - overrides/*.yml

resources:
  jobs:
    j:
      tasks:
        - task_key: base
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "overrides", "10-first.yml"), []byte(`targets:
  dev:
    resources:
      jobs:
        j:
          tasks:
            - task_key: from_first
              max_retries: 1
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "overrides", "20-second.yml"), []byte(`targets:
  dev:
    resources:
      jobs:
        j:
          tasks:
            - task_key: from_second
              max_retries: 2
`), 0o600))

	ctx := logdiag.InitContext(t.Context())
	b, err := bundle.Load(ctx, dir)
	require.NoError(t, err)
	mutator.DefaultMutators(ctx, b)
	bundle.ApplyContext(ctx, b, mutator.SelectTarget("dev"))
	bundle.ApplySeqContext(ctx, b, resourcemutator.MergeJobTasks())
	blocks := newBlockResolver(ctx, b)
	require.NotNil(t, blocks)

	resolved, err := resolveSelectors("resources.jobs.j.tasks[task_key='from_second'].max_retries", b, OperationReplace)
	require.NoError(t, err)

	destination, err := blocks.singleDestination(resolved)
	require.NoError(t, err)
	assert.Equal(t, "20-second.yml", filepath.Base(destination.block.file))
	assert.Equal(t, "resources.jobs.j.tasks[0].max_retries", destination.path.String())
}
