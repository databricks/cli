package config

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestVerifyUniqueResourceIdentifiers(t *testing.T) {
	r := Resources{
		Jobs: map[string]*resources.Job{
			"foo": {
				Paths: paths.Paths{
					ConfigFilePath: "foo.yml",
				},
			},
		},
		Models: map[string]*resources.MlflowModel{
			"bar": {
				Paths: paths.Paths{
					ConfigFilePath: "bar.yml",
				},
			},
		},
		Experiments: map[string]*resources.MlflowExperiment{
			"foo": {
				Paths: paths.Paths{
					ConfigFilePath: "foo2.yml",
				},
			},
		},
	}
	_, err := r.VerifyUniqueResourceIdentifiers()
	assert.ErrorContains(t, err, "multiple resources named foo (job at foo.yml, mlflow_experiment at foo2.yml)")
}

func TestVerifySafeMerge(t *testing.T) {
	r := Resources{
		Jobs: map[string]*resources.Job{
			"foo": {
				Paths: paths.Paths{
					ConfigFilePath: "foo.yml",
				},
			},
		},
		Models: map[string]*resources.MlflowModel{
			"bar": {
				Paths: paths.Paths{
					ConfigFilePath: "bar.yml",
				},
			},
		},
	}
	other := Resources{
		Pipelines: map[string]*resources.Pipeline{
			"foo": {
				Paths: paths.Paths{
					ConfigFilePath: "foo2.yml",
				},
			},
		},
	}
	err := r.VerifySafeMerge(&other)
	assert.ErrorContains(t, err, "multiple resources named foo (job at foo.yml, pipeline at foo2.yml)")
}

func TestVerifySafeMergeForSameResourceType(t *testing.T) {
	r := Resources{
		Jobs: map[string]*resources.Job{
			"foo": {
				Paths: paths.Paths{
					ConfigFilePath: "foo.yml",
				},
			},
		},
		Models: map[string]*resources.MlflowModel{
			"bar": {
				Paths: paths.Paths{
					ConfigFilePath: "bar.yml",
				},
			},
		},
	}
	other := Resources{
		Jobs: map[string]*resources.Job{
			"foo": {
				Paths: paths.Paths{
					ConfigFilePath: "foo2.yml",
				},
			},
		},
	}
	err := r.VerifySafeMerge(&other)
	assert.ErrorContains(t, err, "multiple resources named foo (job at foo.yml, job at foo2.yml)")
}

func TestVerifySafeMergeForRegisteredModels(t *testing.T) {
	r := Resources{
		Jobs: map[string]*resources.Job{
			"foo": {
				Paths: paths.Paths{
					ConfigFilePath: "foo.yml",
				},
			},
		},
		RegisteredModels: map[string]*resources.RegisteredModel{
			"bar": {
				Paths: paths.Paths{
					ConfigFilePath: "bar.yml",
				},
			},
		},
	}
	other := Resources{
		RegisteredModels: map[string]*resources.RegisteredModel{
			"bar": {
				Paths: paths.Paths{
					ConfigFilePath: "bar2.yml",
				},
			},
		},
	}
	err := r.VerifySafeMerge(&other)
	assert.ErrorContains(t, err, "multiple resources named bar (registered_model at bar.yml, registered_model at bar2.yml)")
}

// This test ensures that all resources have a custom marshaller and unmarshaller.
// This is required because DABs resources map to Databricks APIs, and they do so
// by embedding the corresponding Go SDK structs.
// The Go SDK structs implement custom marshalling and unmarshalling methods. If we
// do not implement custom marshalling and unmarshalling methods for the resources
// at the top level, marshalling and unmarshalling will panic.
//
// Why do we not assert this using an interface to assert MarshalJSON and UnmarshalJSON
// are implemented at the top level?
// If a method is implemented for an embedded struct, the top level struct will
// also have that method and satisfy the interface. This is why we cannot assert
// that the methods are implemented at the top level using an interface.
//
// Why don't we use reflection to assert that the methods are implemented at the
// top level?
// Same problem as above, the golang reflection package does not seem to provide
// a way to directly assert that MarshalJSON and UnmarshalJSON are implemented
// at the top level.
func TestCustomMarshallerIsImplemented(t *testing.T) {
	r := Resources{}
	rt := reflect.TypeOf(r)

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)

		// Fields in Resources are expected be of the form map[string]*resourceStruct
		assert.Equal(t, field.Type.Kind(), reflect.Map, "Resource %s is not a map", field.Name)
		kt := field.Type.Key()
		assert.Equal(t, kt.Kind(), reflect.String, "Resource %s is not a map with string keys", field.Name)
		vt := field.Type.Elem()
		assert.Equal(t, vt.Kind(), reflect.Ptr, "Resource %s is not a map with pointer values", field.Name)

		// Marshalling a resourceStruct will panic if resourceStruct does not have a custom marshaller
		// This is because resourceStruct embeds a Go SDK struct that implements
		// a custom marshaller.
		// Eg: resource.Job implements MarshalJSON
		v := reflect.Zero(vt.Elem()).Interface()
		assert.NotPanics(t, func() {
			json.Marshal(v)
		}, "Resource %s does not have a custom marshaller", field.Name)

		// Unmarshalling a *resourceStruct will panic if the resource does not have a custom unmarshaller
		// This is because resourceStruct embeds a Go SDK struct that implements
		// a custom unmarshaller.
		// Eg: *resource.Job implements UnmarshalJSON
		v = reflect.New(vt.Elem()).Interface()
		assert.NotPanics(t, func() {
			json.Unmarshal([]byte("{}"), v)
		}, "Resource %s does not have a custom unmarshaller", field.Name)
	}
}
