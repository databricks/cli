package yamlsaver

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

func SaveAsYAML(data any, filename string, force bool) error {
	// check that file exists
	info, err := os.Stat(filename)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s is a directory", filename)
		}
		if !force {
			return fmt.Errorf("%s already exists. Use --force to overwrite", filename)
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	err = encode(data, file)
	if err != nil {
		return err
	}
	return nil
}

func encode(data any, w io.Writer) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc.Encode(data)
}
