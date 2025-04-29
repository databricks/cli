package apps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/env"
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

// ReadAppSpecFile reads the app spec file and returns the AppSpec struct
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

func (spec *AppSpec) LoadEnvVars(ctx context.Context, customEnv []string) ([]string, error) {
	for _, envVar := range spec.EnvVars {
		if envVar.Value != nil {
			customEnv = append(customEnv, envVar.Name+"="+*envVar.Value)
		}

		if envVar.ValueFrom != nil {
			e, ok := env.Lookup(ctx, envVar.Name)
			if ok {
				customEnv = append(customEnv, envVar.Name+"="+e)
			}
			found := false
			for _, e := range customEnv {
				if strings.HasPrefix(e, envVar.Name+"=") {
					found = true
					break
				}
			}
			if !found {
				return customEnv, fmt.Errorf("%s defined in %s with valueFrom property and can't be resolved locally. "+
					"Please set %s environment variable in your terminal or using --env flag", envVar.Name, spec.fileName, envVar.Name)
			}
		}
	}
	return customEnv, nil
}
