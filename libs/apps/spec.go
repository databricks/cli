package apps

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type AppEnvVar struct {
	Name      string  `yaml:"name"`
	Value     *string `yaml:"value,omitempty"`
	ValueFrom *string `yaml:"valueFrom,omitempty"`
}

// AppSpec is a struct that holds the app spec. It is read from app.yaml
type AppSpec struct {
	fileName string
	config   *Config

	Command []string    `yaml:"command"`
	EnvVars []AppEnvVar `yaml:"env"`
}

func (a *AppSpec) GetFileName() string {
	return a.fileName
}

// readAppSpecFile reads the app spec file and returns the AppSpec struct
func ReadAppSpecFile(config *Config) (*AppSpec, error) {
	spec := &AppSpec{config: config}
	for _, file := range config.AppSpecFiles {
		// Read the yaml file
		yamlFile, err := os.ReadFile(filepath.Join(config.AppPath, file))
		if os.IsNotExist(err) {
			continue
		}

		if err != nil {
			return spec, fmt.Errorf("%s reading error", file)
		}

		err = yaml.Unmarshal(yamlFile, spec)
		if err != nil {
			return spec, err
		}
		spec.fileName = file
		return spec, nil
	}
	return spec, nil
}
