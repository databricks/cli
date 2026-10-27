package convert

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertFromTypedToTypedEqual[T any](t *testing.T, src T) {
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	var dst T
	err = ToTyped(&dst, nv)
	require.NoError(t, err)
	assert.Equal(t, src, dst)
}

func TestAdditional(t *testing.T) {
	type StructType struct {
		Str string `json:"str"`
	}

	type Tmp struct {
		MapToPointer   map[string]*string `json:"map_to_pointer"`
		SliceOfPointer []*string          `json:"slice_of_pointer"`
		NestedStruct   StructType         `json:"nested_struct"`
	}

	t.Run("nil", func(t *testing.T) {
		assertFromTypedToTypedEqual(t, Tmp{})
	})

	t.Run("empty map", func(t *testing.T) {
		assertFromTypedToTypedEqual(t, Tmp{
			MapToPointer: map[string]*string{},
		})
	})

	t.Run("map with empty string value", func(t *testing.T) {
		s := ""
		assertFromTypedToTypedEqual(t, Tmp{
			MapToPointer: map[string]*string{
				"key": &s,
			},
		})
	})

	t.Run("map with nil value", func(t *testing.T) {
		assertFromTypedToTypedEqual(t, Tmp{
			MapToPointer: map[string]*string{
				"key": nil,
			},
		})
	})

	t.Run("empty slice", func(t *testing.T) {
		assertFromTypedToTypedEqual(t, Tmp{
			SliceOfPointer: []*string{},
		})
	})

	t.Run("slice with nil value", func(t *testing.T) {
		assertFromTypedToTypedEqual(t, Tmp{
			SliceOfPointer: []*string{nil},
		})
	})

	t.Run("pointer to a empty string", func(t *testing.T) {
		s := ""
		assertFromTypedToTypedEqual(t, &s)
	})

	t.Run("nil pointer", func(t *testing.T) {
		var s *string
		assertFromTypedToTypedEqual(t, s)
	})

	t.Run("pointer to struct with scalar values", func(t *testing.T) {
		s := ""
		type foo struct {
			A string  `json:"a"`
			B int     `json:"b"`
			C bool    `json:"c"`
			D *string `json:"d"`
		}
		assertFromTypedToTypedEqual(t, &foo{
			A: "a",
			B: 1,
			C: true,
			D: &s,
		})
		assertFromTypedToTypedEqual(t, &foo{
			A: "",
			B: 0,
			C: false,
			D: nil,
		})
	})

	t.Run("map with scalar values", func(t *testing.T) {
		assertFromTypedToTypedEqual(t, map[string]string{
			"a": "a",
			"b": "b",
			"c": "",
		})
		assertFromTypedToTypedEqual(t, map[string]int{
			"a": 1,
			"b": 0,
			"c": 2,
		})
	})
}

func TestEndToEndForceSendFields(t *testing.T) {
	type Inner struct {
		InnerField      string   `json:"inner_field"`
		ForceSendFields []string `json:"-"`
	}
	type Outer struct {
		OuterField string `json:"outer_field"`
		Inner
	}

	// Test with zero value in embedded struct
	src := Outer{
		OuterField: "outer_value",
		Inner: Inner{
			InnerField:      "",                     // Zero value
			ForceSendFields: []string{"InnerField"}, // Should be preserved
		},
	}

	assertFromTypedToTypedEqual(t, src)
}

func TestEndToEndPointerForceSendFields(t *testing.T) {
	type NewCluster struct {
		NumWorkers      int      `json:"num_workers"`
		SparkVersion    string   `json:"spark_version"`
		ForceSendFields []string `json:"-"`
	}
	type JobCluster struct {
		JobClusterKey string      `json:"job_cluster_key"`
		NewCluster    *NewCluster `json:"new_cluster"`
	}
	type JobSettings struct {
		JobClusters     []JobCluster `json:"job_clusters"`
		Name            string       `json:"name"`
		ForceSendFields []string     `json:"-"`
	}

	// Test with zero value in pointer embedded struct (like acceptance test)
	src := JobSettings{
		Name: "test-job",
		JobClusters: []JobCluster{
			{
				JobClusterKey: "key",
				NewCluster: &NewCluster{
					NumWorkers:      0, // Zero value
					SparkVersion:    "13.3.x-scala2.12",
					ForceSendFields: []string{"NumWorkers"}, // Should be preserved
				},
			},
		},
	}

	assertFromTypedToTypedEqual(t, src)
}
