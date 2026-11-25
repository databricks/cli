package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceTypeRegistryOrdersByResourceType(t *testing.T) {
	r := NewResourceTypeRegistry[string]("value")
	r.Register("dashboards", "d")
	r.Register("pipelines", "p")
	r.Register("jobs", "j")

	var keys, values []string
	for k, v := range r.All() {
		keys = append(keys, k)
		values = append(values, v)
	}
	assert.Equal(t, []string{"jobs", "pipelines", "dashboards"}, keys)
	assert.Equal(t, []string{"j", "p", "d"}, values)
}

func TestResourceTypeRegistryGet(t *testing.T) {
	r := NewResourceTypeRegistry[string]("value")
	r.Register("jobs", "j")

	v, ok := r.Get("jobs")
	assert.True(t, ok)
	assert.Equal(t, "j", v)

	_, ok = r.Get("pipelines")
	assert.False(t, ok)
}

func TestResourceTypeRegistryRejectsUnknownType(t *testing.T) {
	r := NewResourceTypeRegistry[string]("value")
	r.Register("nope", "n")
	assert.PanicsWithValue(t, `value registered for unknown resource type "nope"`, func() {
		r.Get("jobs")
	})
}

func TestResourceTypeRegistryRejectsDuplicate(t *testing.T) {
	r := NewResourceTypeRegistry[string]("value")
	r.Register("jobs", "a")
	r.Register("jobs", "b")
	assert.PanicsWithValue(t, `value registered more than once for resource type "jobs"`, func() {
		r.Get("jobs")
	})
}

func TestResourceTypeRegistryRejectsRegisterAfterFirstUse(t *testing.T) {
	r := NewResourceTypeRegistry[string]("value")
	r.Register("jobs", "j")
	r.Get("jobs")
	assert.PanicsWithValue(t, `value registered for resource type "pipelines" after first use`, func() {
		r.Register("pipelines", "p")
	})
}
