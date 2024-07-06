package yamlloader

import (
	"io"

	"github.com/databricks/cli/libs/dyn"
	"gopkg.in/yaml.v3"
)

func LoadYAML(path string, r io.Reader) (dyn.Value, error) {
	var node yaml.Node
	dec := yaml.NewDecoder(r)
	err := dec.Decode(&node)
	if err != nil {
		if err == io.EOF {
			return dyn.NilValue, nil
		}
		return dyn.InvalidValue, err
	}

	return newLoader(path).load(&node)
}
