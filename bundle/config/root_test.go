package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/databricks/bricks/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootMarshalUnmarshal(t *testing.T) {
	// Marshal empty
	buf, err := json.Marshal(&Root{})
	require.NoError(t, err)

	// Unmarshal empty
	var root Root
	err = json.Unmarshal(buf, &root)
	require.NoError(t, err)

	// Compare
	assert.True(t, reflect.DeepEqual(Root{}, root))
}

func TestRootLoad(t *testing.T) {
	root := &Root{}
	err := root.Load("../tests/basic/bundle.yml")
	require.NoError(t, err)
	assert.Equal(t, "basic", root.Bundle.Name)
}

func TestRootMergeStruct(t *testing.T) {
	root := &Root{
		Path: "path",
		Workspace: Workspace{
			Host:    "foo",
			Profile: "profile",
		},
	}
	other := &Root{
		Path: "path",
		Workspace: Workspace{
			Host: "bar",
		},
	}
	assert.NoError(t, root.Merge(other))
	assert.Equal(t, "bar", root.Workspace.Host)
	assert.Equal(t, "profile", root.Workspace.Profile)
}

func TestRootMergeMap(t *testing.T) {
	root := &Root{
		Path: "path",
		Environments: map[string]*Environment{
			"development": {
				Workspace: &Workspace{
					Host:    "foo",
					Profile: "profile",
				},
			},
		},
	}
	other := &Root{
		Path: "path",
		Environments: map[string]*Environment{
			"development": {
				Workspace: &Workspace{
					Host: "bar",
				},
			},
		},
	}
	assert.NoError(t, root.Merge(other))
	assert.Equal(t, &Workspace{Host: "bar", Profile: "profile"}, root.Environments["development"].Workspace)
}

func TestDuplicateIdOnLoadReturnsError(t *testing.T) {
	dir := t.TempDir()
	root := &Root{
		Resources: Resources{
			Jobs: map[string]*resources.Job{
				"foo": {},
				"bar": {},
			},
			Pipelines: map[string]*resources.Pipeline{
				"foo": {},
			},
		},
	}
	b, err := json.Marshal(root)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "bundle.yml"), b, os.ModePerm)
	require.NoError(t, err)

	err = root.Load(filepath.Join(dir, "bundle.yml"))
	assert.ErrorContains(t, err, "duplicate identifier foo")
}

func TestRootMergeDetectsDuplicateIds(t *testing.T) {
	root := &Root{
		Resources: Resources{
			Jobs: map[string]*resources.Job{
				"foo": {},
			},
			Pipelines: map[string]*resources.Pipeline{
				"bar": {},
			},
		},
	}
	other := &Root{
		Resources: Resources{
			Models: map[string]*resources.MlflowModel{
				"bar": {},
			},
		},
	}
	err := root.Merge(other)
	assert.ErrorContains(t, err, "duplicate identifier bar")
}
