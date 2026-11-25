package configsync

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/stretchr/testify/assert"
)

func TestChangeHooksApplyOnlyToTheirResourceType(t *testing.T) {
	all := func(string, *deployplan.ChangeDesc) bool { return true }
	registry := changeHookRegistry{}
	registry.register("pipelines", changeHook{
		keepSkipped:         all,
		syncsBackendDefault: func(string) bool { return true },
		newSkip: func(context.Context, *deployplan.Plan, map[string][]dresources.StatePermission) changeSkip {
			return func(string, string, *deployplan.ChangeDesc) bool { return true }
		},
	})
	skip := registry.newSkip(t.Context(), &deployplan.Plan{}, nil)
	cd := &deployplan.ChangeDesc{}

	assert.True(t, registry.keepSkipped("resources.pipelines.p", "resources.pipelines.p.name", cd))
	assert.True(t, registry.syncsBackendDefault("resources.pipelines.p", "resources.pipelines.p.name"))
	assert.True(t, skip("resources.pipelines.p", "resources.pipelines.p.name", cd))

	assert.False(t, registry.keepSkipped("resources.jobs.j", "resources.jobs.j.name", cd))
	assert.False(t, registry.syncsBackendDefault("resources.jobs.j", "resources.jobs.j.name"))
	assert.False(t, skip("resources.jobs.j", "resources.jobs.j.name", cd))
}

func TestChangeHookFieldsAreOptional(t *testing.T) {
	registry := changeHookRegistry{}
	registry.register("jobs", changeHook{})
	skip := registry.newSkip(t.Context(), &deployplan.Plan{}, nil)
	cd := &deployplan.ChangeDesc{}

	assert.False(t, registry.keepSkipped("resources.jobs.j", "resources.jobs.j.run_as", cd))
	assert.False(t, registry.syncsBackendDefault("resources.jobs.j", "resources.jobs.j.run_as"))
	assert.False(t, skip("resources.jobs.j", "resources.jobs.j.run_as", cd))
}
