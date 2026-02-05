package sdkdocs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadIndex(t *testing.T) {
	index, err := LoadIndex()
	require.NoError(t, err)
	require.NotNil(t, index)

	// Verify the index has expected structure
	assert.NotEmpty(t, index.Version)
	assert.NotEmpty(t, index.GeneratedAt)
	assert.NotEmpty(t, index.Services)

	// Check that jobs service exists and has methods
	jobsService := index.GetService("jobs")
	require.NotNil(t, jobsService, "jobs service should exist")
	assert.Equal(t, "Jobs", jobsService.Name)
	assert.NotEmpty(t, jobsService.Methods)

	// Check that Create method exists
	createMethod := index.GetMethod("jobs", "Create")
	require.NotNil(t, createMethod, "jobs.Create method should exist")
	assert.Equal(t, "Create", createMethod.Name)
	assert.NotEmpty(t, createMethod.Description)
}

func TestGetMethod(t *testing.T) {
	index, err := LoadIndex()
	require.NoError(t, err)

	t.Run("existing method", func(t *testing.T) {
		method := index.GetMethod("jobs", "Create")
		require.NotNil(t, method)
		assert.Equal(t, "Create", method.Name)
	})

	t.Run("non-existing method", func(t *testing.T) {
		method := index.GetMethod("jobs", "NonExistent")
		assert.Nil(t, method)
	})

	t.Run("non-existing service", func(t *testing.T) {
		method := index.GetMethod("nonexistent", "Create")
		assert.Nil(t, method)
	})
}

func TestGetService(t *testing.T) {
	index, err := LoadIndex()
	require.NoError(t, err)

	t.Run("existing service", func(t *testing.T) {
		service := index.GetService("jobs")
		require.NotNil(t, service)
		assert.Equal(t, "Jobs", service.Name)
	})

	t.Run("non-existing service", func(t *testing.T) {
		service := index.GetService("nonexistent")
		assert.Nil(t, service)
	})
}

func TestListServices(t *testing.T) {
	index, err := LoadIndex()
	require.NoError(t, err)

	services := index.ListServices()
	assert.NotEmpty(t, services)
	assert.Contains(t, services, "jobs")
}

func TestGetEnum(t *testing.T) {
	index, err := LoadIndex()
	require.NoError(t, err)

	t.Run("existing enum", func(t *testing.T) {
		enum := index.GetEnum("jobs.RunLifeCycleState")
		require.NotNil(t, enum)
		assert.Equal(t, "RunLifeCycleState", enum.Name)
		assert.NotEmpty(t, enum.Values)
	})

	t.Run("non-existing enum", func(t *testing.T) {
		enum := index.GetEnum("nonexistent.Enum")
		assert.Nil(t, enum)
	})
}
