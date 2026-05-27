package config_test

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
)

func TestBindIsEmpty(t *testing.T) {
	cases := []struct {
		name string
		bind config.Bind
		want bool
	}{
		{"nil", nil, true},
		{"empty outer", config.Bind{}, true},
		{"empty inner", config.Bind{"jobs": {}}, true},
		{"populated", config.Bind{"jobs": {"foo": {ID: "1"}}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.bind.IsEmpty())
		})
	}
}

func TestBindForEachIteratesInSortedOrder(t *testing.T) {
	bind := config.Bind{
		"pipelines": {"baz": {ID: "3"}},
		"jobs":      {"foo": {ID: "1"}, "bar": {ID: "2"}},
	}
	type entry struct{ rt, rn, id string }
	var got []entry
	bind.ForEach(func(rt, rn, id string) {
		got = append(got, entry{rt, rn, id})
	})
	// ForEach guarantees stable order so multi-error diagnostics are reproducible.
	assert.Equal(t, []entry{
		{"jobs", "bar", "2"},
		{"jobs", "foo", "1"},
		{"pipelines", "baz", "3"},
	}, got)
}

func TestBindValidate(t *testing.T) {
	cases := []struct {
		name      string
		bind      config.Bind
		wantError bool
	}{
		{"top-level resource", config.Bind{"jobs": {"foo": {ID: "1"}}}, false},
		{"permissions child", config.Bind{"jobs.permissions": {"foo": {ID: "1"}}}, true},
		{"grants child", config.Bind{"schemas.grants": {"foo": {ID: "1"}}}, true},
		// Substring match must NOT trigger; only the .permissions / .grants suffix does.
		{"name containing permissions", config.Bind{"my_permissions_jobs": {"foo": {ID: "1"}}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			diags := c.bind.Validate()
			if c.wantError {
				assert.True(t, diags.HasError())
				assert.Equal(t, diag.Error, diags[0].Severity)
			} else {
				assert.False(t, diags.HasError())
			}
		})
	}
}
