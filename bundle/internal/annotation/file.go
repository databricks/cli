package annotation

import (
	"bytes"
	"os"
	"slices"

	yaml3 "gopkg.in/yaml.v3"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
)

// Parsed file with annotations, expected format:
// github.com/databricks/cli/bundle/config.Bundle:
//
//	cluster_id:
//	   description: "Description"
type File map[string]map[string]Descriptor

// Load loads annotations from a single file.
func Load(path string) (File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(File), nil
		}
		return nil, err
	}

	dynVal, err := yamlloader.LoadYAML(path, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}

	var data File
	if err := convert.ToTyped(&data, dynVal); err != nil {
		return nil, err
	}
	if data == nil {
		data = make(File)
	}
	return data, nil
}

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

// Save saves the annotation file to the given path.
func (f File) Save(path string) error {
	annotationOrder := yamlsaver.NewOrder([]string{"description", "markdown_description", "title", "default", "enum", "since_version"})
	style := map[string]yaml3.Style{}

	order := alphabeticalOrder(f)
	dynMap := map[string]dyn.Value{}
	for k, v := range f {
		style[k] = yaml3.LiteralStyle

		properties := map[string]dyn.Value{}
		propertiesOrder := alphabeticalOrder(v)
		for key, value := range v {
			d, err := convert.FromTyped(value, dyn.NilValue)
			if d.Kind() == dyn.KindNil || err != nil {
				properties[key] = dyn.NewValue(map[string]dyn.Value{}, []dyn.Location{{Line: propertiesOrder.Get(key)}})
				continue
			}
			val, err := yamlsaver.ConvertToMapValue(value, annotationOrder, []string{}, map[string]dyn.Value{})
			if err != nil {
				return err
			}
			properties[key] = val.WithLocations([]dyn.Location{{Line: propertiesOrder.Get(key)}})
		}

		dynMap[k] = dyn.NewValue(properties, []dyn.Location{{Line: order.Get(k)}})
	}

	saver := yamlsaver.NewSaverWithStyle(style)
	return saver.SaveAsYAML(dynMap, path, true)
}

func alphabeticalOrder[T any](mapping map[string]T) *yamlsaver.Order {
	var order []string
	for k := range mapping {
		order = append(order, k)
	}
	slices.Sort(order)
	return yamlsaver.NewOrder(order)
}
