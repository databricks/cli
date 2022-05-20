package python

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"golang.org/x/mod/semver"
)

type Dependency struct {
	Name     string
	Operator string
	Version  string
	Location string // @ file:///usr/loca
}

func (d Dependency) CanonicalVersion() string {
	return semver.Canonical(fmt.Sprintf("v%s", d.Version))
}

type Environment []Dependency

func (e Environment) Has(name string) bool {
	for _, d := range e {
		if d.Name == name {
			return true
		}
	}
	return false
}

func Freeze(ctx context.Context) (Environment, error) {
	out, err := Py(ctx, "-m", "pip", "freeze")
	if err != nil {
		return nil, err
	}
	env := Environment{}
	deps := strings.Split(out, "\n")
	for _, raw := range deps {
		env = append(env, DependencyFromSpec(raw))
	}
	return env, nil
}

func DependencyFromSpec(raw string) (d Dependency) {
	// TODO: write a normal parser for this
	rawSplit := strings.Split(raw, "==")
	if len(rawSplit) != 2 {
		log.Printf("[DEBUG] Skipping invalid dep: %s", raw)
		return
	}
	d.Name = rawSplit[0]
	d.Operator = "=="
	d.Version = rawSplit[1]
	return
}

type Distribution struct {
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Packages        []string `json:"packages"`
	InstallRequires []string `json:"install_requires,omitempty"`
}

// InstallEnvironment returns only direct install dependencies
func (d Distribution) InstallEnvironment() (env Environment) {
	for _, raw := range d.InstallRequires {
		env = append(env, DependencyFromSpec(raw))
	}
	return
}

// ReadDistribution "parses" metadata from setup.py file.
func ReadDistribution(ctx context.Context) (d Distribution, err error) {
	out, err := PyInline(ctx, `
	import setuptools, json, sys
	setup_config = {} # actual args for setuptools.dist.Distribution
	def capture(**kwargs): global setup_config; setup_config = kwargs
	setuptools.setup = capture
	import setup
	json.dump(setup_config, sys.stdout)`)
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(out), &d)
	return
}
