package metadata

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeMetadataWalk(t *testing.T) {
	type foo struct {
		privateField              int `bundle:"metadata"`
		IgnoredForMetadata        string
		IncludedInMetadata        string  `bundle:"metadata"`
		PointerIncludedInMetadata *string `bundle:"readonly,metadata"`
	}

	myString := "hello"

	config := foo{
		privateField:              2,
		IgnoredForMetadata:        "abc",
		IncludedInMetadata:        "xyz",
		PointerIncludedInMetadata: &myString,
	}
	metadata := foo{}
	err := walk(reflect.ValueOf(config), reflect.ValueOf(&metadata).Elem())
	assert.NoError(t, err)
	assert.Equal(t, foo{IncludedInMetadata: "xyz", PointerIncludedInMetadata: &myString}, metadata)
}

func TestComputeMetadataRecursivelyWalkAnonymousStruct(t *testing.T) {
	type Foo struct {
		privateField       int `bundle:"metadata"`
		IgnoredForMetadata string
		IncludedInMetadata string `bundle:"metadata"`
	}
	type bar struct {
		Apple int `bundle:"readonly,metadata"`
		Foo
		Guava string
	}

	config := bar{
		Apple: 123,
		Foo: Foo{
			privateField:       2,
			IgnoredForMetadata: "abc",
			IncludedInMetadata: "xyz",
		},
		Guava: "guava",
	}

	metadata := bar{}
	err := walk(reflect.ValueOf(config), reflect.ValueOf(&metadata).Elem())
	assert.NoError(t, err)
	assert.Equal(t, bar{
		Foo: Foo{
			IncludedInMetadata: "xyz",
		},
		Apple: 123,
	}, metadata)
}

func TestComputeMetadataRecursivelyWalkAnonymousStructPointer(t *testing.T) {
	type Foo struct {
		privateField       int `bundle:"metadata"`
		IgnoredForMetadata string
		IncludedInMetadata string `bundle:"metadata"`
	}
	type bar struct {
		Apple int `bundle:"readonly,metadata"`
		*Foo
		Guava string
	}

	config := bar{
		Apple: 123,
		Foo: &Foo{
			privateField:       2,
			IgnoredForMetadata: "abc",
			IncludedInMetadata: "xyz",
		},
		Guava: "guava",
	}

	metadata := bar{}
	err := walk(reflect.ValueOf(config), reflect.ValueOf(&metadata).Elem())
	assert.NoError(t, err)
	assert.Equal(t, bar{
		Foo: &Foo{
			IncludedInMetadata: "xyz",
		},
		Apple: 123,
	}, metadata)
}

func TestComputeMetadataRecursivelyWalkStruct(t *testing.T) {
	type Foo struct {
		privateField       int `bundle:"metadata"`
		IgnoredForMetadata string
		IncludedInMetadata string `bundle:"metadata"`
	}
	type bar struct {
		Apple int `bundle:"readonly,metadata"`
		Mango Foo
		Guava string
	}

	config := bar{
		Apple: 123,
		Mango: Foo{
			privateField:       2,
			IgnoredForMetadata: "abc",
			IncludedInMetadata: "xyz",
		},
		Guava: "guava",
	}

	metadata := bar{}
	err := walk(reflect.ValueOf(config), reflect.ValueOf(&metadata).Elem())
	assert.NoError(t, err)
	assert.Equal(t, bar{
		Mango: Foo{
			IncludedInMetadata: "xyz",
		},
		Apple: 123,
	}, metadata)
}

func TestComputeMetadataRecursivelyWalkStructPointer(t *testing.T) {
	type Foo struct {
		privateField       int `bundle:"metadata"`
		IgnoredForMetadata string
		IncludedInMetadata string `bundle:"metadata"`
	}
	type bar struct {
		Apple int `bundle:"readonly,metadata"`
		Mango *Foo
		Guava string
	}

	config := bar{
		Apple: 123,
		Mango: &Foo{
			privateField:       2,
			IgnoredForMetadata: "abc",
			IncludedInMetadata: "xyz",
		},
		Guava: "guava",
	}

	metadata := bar{}
	err := walk(reflect.ValueOf(config), reflect.ValueOf(&metadata).Elem())
	assert.NoError(t, err)
	assert.Equal(t, bar{
		Mango: &Foo{
			IncludedInMetadata: "xyz",
		},
		Apple: 123,
	}, metadata)
}

func TestComputeMetadataMutator(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name:   "my-bundle",
				Target: "development",
				Git: config.Git{
					Branch:    "my-branch",
					OriginURL: "www.host.com",
					Commit:    "abcd",
				},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my-job-1": {
						Paths: paths.Paths{
							ConfigFilePath: "a/b/c",
						},
						JobSettings: &jobs.JobSettings{
							Name: "My Job One",
						},
					},
					"my-job-2": {
						Paths: paths.Paths{
							ConfigFilePath: "d/e/f",
						},
						JobSettings: &jobs.JobSettings{
							Name: "My Job Two",
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"my-pipeline": {
						Paths: paths.Paths{
							ConfigFilePath: "abc",
						},
					},
				},
			},
		},
	}

	// TODO: make the structs pointers to clean them up
	// TODO: assert on the raw text output in that case
	expectedMetadata := deploy.Metadata{
		Version: deploy.LatestMetadataVersion,
		Config: config.Root{
			Bundle: config.Bundle{
				Git: config.Git{
					Branch:    "my-branch",
					OriginURL: "www.host.com",
					Commit:    "abcd",
				},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my-job-1": {
						Paths: paths.Paths{
							ConfigFilePath: "a/b/c",
						},
						JobSettings: &jobs.JobSettings{},
					},
					"my-job-2": {
						Paths: paths.Paths{
							ConfigFilePath: "d/e/f",
						},
						JobSettings: &jobs.JobSettings{},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"my-pipeline": {
						Paths: paths.Paths{
							ConfigFilePath: "",
						},
					},
				},
			},
		},
	}

	err := ComputeMetadata().Apply(context.Background(), b)
	require.NoError(t, err)

	// Print some text for debugging
	actual, err := json.MarshalIndent(b.Metadata, "		", "	")
	assert.NoError(t, err)
	t.Log("[DEBUG] actual: ", string(actual))
	expected, err := json.MarshalIndent(expectedMetadata, "		", "	")
	assert.NoError(t, err)
	t.Log("[DEBUG] expected: ", string(expected))

	assert.Equal(t, expectedMetadata, b.Metadata)
}
