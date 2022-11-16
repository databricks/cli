package config

import (
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/imdario/mergo"
)

// FileName is the name of bundle configuration file.
const FileName = "bundle.yml"

type Root struct {
	Path string `json:"-"`

	Bundle Bundle `json:"bundle"`

	Include []string `json:"include,omitempty"`

	Workspace Workspace `json:"workspace"`

	Resources Resources `json:"resources"`

	// Environments can be used to differentiate settings and resources between
	// bundle deployment environments (e.g. development, staging, production).
	// If not specified, the code below initializes this field with a
	// single default-initialized environment called "default".
	Environments map[string]*Environment `json:"environments,omitempty"`
}

func Load(path string) (*Root, error) {
	var r Root

	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// If we were given a directory, assume this is the bundle root.
	if stat.IsDir() {
		r.Path = path
		path = filepath.Join(path, FileName)
	}

	if err := r.Load(path); err != nil {
		return nil, err
	}

	return &r, nil
}

func (r *Root) Load(file string) error {
	raw, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(raw, r)
	if err != nil {
		return err
	}
	return nil
}

func (r *Root) Merge(other *Root) error {
	// TODO: define and test semantics for merging.
	return mergo.MergeWithOverwrite(r, other)
}

func (r *Root) MergeEnvironment(env *Environment) error {
	var err error

	if env.Bundle != nil {
		err = mergo.MergeWithOverwrite(&r.Bundle, env.Bundle)
		if err != nil {
			return err
		}
	}

	if env.Workspace != nil {
		err = mergo.MergeWithOverwrite(&r.Workspace, env.Workspace)
		if err != nil {
			return err
		}
	}

	if env.Resources != nil {
		err = mergo.MergeWithOverwrite(&r.Resources, env.Resources)
		if err != nil {
			return err
		}
	}

	return nil
}
