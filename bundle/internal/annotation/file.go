package annotation

import (
	"bytes"
	"os"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
)

// Parsed file with annotations, expected format:
// github.com/databricks/cli/bundle/config.Bundle:
//
//	cluster_id:
//	   description: "Description"
type File map[string]map[string]Descriptor

func LoadAndMerge(sources []string) (File, error) {
	prev := dyn.NilValue
	for _, path := range sources {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		generated, err := yamlloader.LoadYAML(path, bytes.NewBuffer(b))
		if err != nil {
			return nil, err
		}
		prev, err = merge.Merge(prev, generated)
		if err != nil {
			return nil, err
		}
	}

	var data File

	err := convert.ToTyped(&data, prev)
	if err != nil {
		return nil, err
	}
	return data, nil
}
