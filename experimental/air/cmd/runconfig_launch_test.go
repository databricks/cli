package aircmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunConfigTimeoutSeconds(t *testing.T) {
	c := &runConfig{}
	assert.Equal(t, 0, c.timeoutSeconds())

	c.TimeoutMinutes = new(2)
	assert.Equal(t, 120, c.timeoutSeconds())
}

func TestRunConfigMaxRetries(t *testing.T) {
	c := &runConfig{}
	assert.Equal(t, defaultMaxRetries, c.maxRetries())

	c.MaxRetries = new(0)
	assert.Equal(t, 0, c.maxRetries())

	c.MaxRetries = new(7)
	assert.Equal(t, 7, c.maxRetries())
}

func TestRunConfigDockerImageURL(t *testing.T) {
	c := &runConfig{}
	assert.Empty(t, c.dockerImageURL())

	c.Environment = &environmentConfig{}
	assert.Empty(t, c.dockerImageURL())

	c.Environment.DockerImage = &dockerImageConfig{URL: "org/repo:tag"}
	assert.Equal(t, "org/repo:tag", c.dockerImageURL())
}

func TestRunConfigDependencies(t *testing.T) {
	t.Run("unset", func(t *testing.T) {
		c := &runConfig{}
		_, ok := c.requirementsFile()
		assert.False(t, ok)
		_, ok = c.inlineDependencies()
		assert.False(t, ok)
	})

	t.Run("file path", func(t *testing.T) {
		c := &runConfig{Environment: &environmentConfig{
			Dependencies: dependencies{set: true, isList: false, path: "req.yaml"},
		}}
		path, ok := c.requirementsFile()
		assert.True(t, ok)
		assert.Equal(t, "req.yaml", path)
		_, ok = c.inlineDependencies()
		assert.False(t, ok)
	})

	t.Run("inline list", func(t *testing.T) {
		c := &runConfig{Environment: &environmentConfig{
			Dependencies: dependencies{set: true, isList: true, list: []string{"torch", "numpy"}},
		}}
		list, ok := c.inlineDependencies()
		assert.True(t, ok)
		assert.Equal(t, []string{"torch", "numpy"}, list)
		_, ok = c.requirementsFile()
		assert.False(t, ok)
	})
}

func TestRunConfigRuntimeVersion(t *testing.T) {
	c := &runConfig{}
	_, ok := c.runtimeVersion()
	assert.False(t, ok)

	c.Environment = &environmentConfig{Version: stringOrInt{set: true, raw: "5"}}
	v, ok := c.runtimeVersion()
	assert.True(t, ok)
	assert.Equal(t, "5", v)
}
