package yamlloader

import (
	"io"

	"github.com/databricks/cli/libs/config"
	"gopkg.in/yaml.v3"
)

func LoadYAML(path string, r io.Reader) (config.Value, error) {
	var node yaml.Node
	dec := yaml.NewDecoder(r)
	err := dec.Decode(&node)
	if err != nil {
		return config.NilValue, err
	}

	return newLoader(path).load(&node)
}
