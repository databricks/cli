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

// loadBundle loads files the way the sync command does, so a target's overrides are
// merged into the resources tree and keyed sequences are merged by key. Keyed
// sequences merge in the initialize phase, after the target is selected, so that
// mutator has to be applied explicitly here.
func loadBundle(t *testing.T, target string, files map[string]string) (*bundle.Bundle, *blockResolver) {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}

	ctx := logdiag.InitContext(t.Context())
	b, err := bundle.Load(ctx, dir)
	require.NoError(t, err)
	mutator.DefaultMutators(ctx, b)
	if target != "" {
		bundle.ApplyContext(ctx, b, mutator.SelectTarget(target))
	}
	bundle.ApplySeqContext(ctx, b, resourcemutator.MergeJobTasks())

	blocks := newBlockResolver(ctx, b)
	require.NotNil(t, blocks)
	return b, blocks
}

// "shared" is defined in both blocks, and the target contributes a task that sorts
// ahead of it, so the merged order differs from either block's own order.
const splitTasks = `bundle:
  name: split

resources:
  jobs:
    j:
      tasks:
        - task_key: shared
          max_retries: 1
          depends_on:
            - task_key: zzz_top
        - task_key: zzz_top
          max_retries: 5

targets:
  dev:
    resources:
      jobs:
        j:
          tasks:
            - task_key: aaa_dev
              max_retries: 6
            - task_key: shared
              timeout_seconds: 60
`

func TestBlockResolverRoutesFieldToDefiningBlock(t *testing.T) {
	b, blocks := loadBundle(t, "dev", map[string]string{"databricks.yml": splitTasks})

	tests := []struct {
		name     string
		path     string
		override bool
		want     string
	}{
		{
			name:     "field defined only top-level",
			path:     "resources.jobs.j.tasks[task_key='zzz_top'].max_retries",
			override: false,
			want:     "resources.jobs.j.tasks[1].max_retries",
		},
		{
			name:     "field defined only in the target",
			path:     "resources.jobs.j.tasks[task_key='aaa_dev'].max_retries",
			override: true,
			want:     "resources.jobs.j.tasks[0].max_retries",
		},
		{
			// The target's value is the deployed one, so writing the top-level copy
			// would leave the effective value unchanged.
			name:     "field of a two-block element routes to its own block",
			path:     "resources.jobs.j.tasks[task_key='shared'].timeout_seconds",
			override: true,
			want:     "resources.jobs.j.tasks[1].timeout_seconds",
		},
		{
			// "shared" is merged index 1 but index 0 in the top-level block, so the
			// enclosing index has to be translated before the nested lookup.
			name:     "nested sequence under a shifted element",
			path:     "resources.jobs.j.tasks[task_key='shared'].depends_on[task_key='zzz_top']",
			override: false,
			want:     "resources.jobs.j.tasks[0].depends_on[0]",
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
	b, blocks := loadBundle(t, "dev", map[string]string{"databricks.yml": splitTasks})

	// Removing an element defined in both blocks has to reach the part in each, and
	// each part has its own index inside its own block.
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
		true:  "resources.jobs.j.tasks[1]",
	}, got)
}

func TestBlockResolverPlacesNewNestedElementInParentBlock(t *testing.T) {
	b, blocks := loadBundle(t, "dev", map[string]string{"databricks.yml": splitTasks})

	// A new depends_on entry on a task whose merged index differs from its
	// block-local one: finding the receiving sequence needs the enclosing index
	// translated first.
	resolved, err := resolveSelectors("resources.jobs.j.tasks[task_key='shared'].depends_on[task_key='aaa_dev']", b, OperationAdd)
	require.NoError(t, err)

	destination, err := blocks.singleDestination(resolved)
	require.NoError(t, err)
	assert.False(t, destination.block.override)
	assert.Equal(t, "resources.jobs.j.tasks[0].depends_on[*]", destination.path.String())
}

func TestBlockResolverRefusesElementSpanningBlocks(t *testing.T) {
	b, blocks := loadBundle(t, "dev", map[string]string{"databricks.yml": splitTasks})

	// A field the element does not define anywhere has no location to route by, so
	// it is an addition and goes to the block declaring the resource.
	resolved, err := resolveSelectors("resources.jobs.j.tasks[task_key='shared'].min_retry_interval_millis", b, OperationReplace)
	require.NoError(t, err)

	destination, err := blocks.singleDestination(resolved)
	require.NoError(t, err)
	assert.False(t, destination.block.override)
}

func TestBlockResolverCountsIndexPerBlockAcrossFiles(t *testing.T) {
	// One target's override spread over two included files. Resource keys are unique
	// across top-level files but that is not enforced inside targets, so a task
	// second in the concatenated target region can be first in the file that defines
	// it -- and the write has to use the latter.
	b, blocks := loadBundle(t, "dev", map[string]string{
		"databricks.yml": `bundle:
  name: split

include:
  - overrides/*.yml

resources:
  jobs:
    j:
      tasks:
        - task_key: base
`,
		"overrides/10-first.yml": `targets:
  dev:
    resources:
      jobs:
        j:
          tasks:
            - task_key: from_first
              max_retries: 1
`,
		"overrides/20-second.yml": `targets:
  dev:
    resources:
      jobs:
        j:
          tasks:
            - task_key: from_second
              max_retries: 2
`,
	})

	resolved, err := resolveSelectors("resources.jobs.j.tasks[task_key='from_second'].max_retries", b, OperationReplace)
	require.NoError(t, err)

	destination, err := blocks.singleDestination(resolved)
	require.NoError(t, err)
	assert.Equal(t, "20-second.yml", filepath.Base(destination.block.file))
	assert.Equal(t, "resources.jobs.j.tasks[0].max_retries", destination.path.String())
}

func TestCompareBlocksOrdersTopLevelFirst(t *testing.T) {
	// blocksOf and sortedBlocks both rely on this order so that declaringBlock picks
	// the top-level block and callers never depend on map iteration order.
	top := sourceBlock{file: "z.yml"}
	override := sourceBlock{override: true, file: "a.yml"}
	assert.Negative(t, compareBlocks(top, override))
	assert.Positive(t, compareBlocks(override, top))
	assert.Negative(t, compareBlocks(sourceBlock{file: "a.yml"}, sourceBlock{file: "b.yml"}))
	assert.Zero(t, compareBlocks(top, top))
}
