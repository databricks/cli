package config

import (
	"encoding/json"
	"reflect"
	"testing"

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
	err := root.Load("./tests/basic/bundle.yml")
	require.NoError(t, err)
	assert.Equal(t, "basic", root.Bundle.Name)
}

func TestRootMergeStruct(t *testing.T) {
	root := &Root{
		Workspace: Workspace{
			Host:    "foo",
			Profile: "profile",
		},
	}
	other := &Root{
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
