package py

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/databricks/bricks/python"
	"golang.org/x/mod/semver"
)

type Dependency struct {
	Name     string
	Operator string
	Version  string
	Location string // @ file:///usr/local
}

func (d Dependency) NormalizedName() string {
	return strings.ToLower(d.Name)
}

func (d Dependency) CanonicalVersion() string {
	return semver.Canonical(fmt.Sprintf("v%s", d.Version))
}

type Environment []Dependency

func (e Environment) Has(name string) bool {
	dep := DependencyFromSpec(name)
	for _, d := range e {
		if d.NormalizedName() == dep.NormalizedName() {
			return true
		}
	}
	return false
}

func DependencyFromSpec(raw string) (d Dependency) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	// TODO: write a normal parser for this
	// see https://peps.python.org/pep-0508/#grammar
	rawSplit := strings.Split(raw, "==")
	if len(rawSplit) != 2 {
		log.Printf("[DEBUG] Skipping invalid dep: %s", raw)
		return Dependency{
			Name: raw,
		}
	}
	d.Name = rawSplit[0]
	d.Operator = "=="
	d.Version = rawSplit[1]
	return
}

// Distribution holds part of PEP426 metadata
// See https://peps.python.org/pep-0426/
type Distribution struct {
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Packages        []string `json:"packages"`
	InstallRequires []string `json:"install_requires,omitempty"`
	TestsRequire    []string `json:"tests_require,omitempty"`
}

// InstallEnvironment returns only direct install dependencies
func (d Distribution) InstallEnvironment() (env Environment) {
	for _, raw := range d.InstallRequires {
		env = append(env, DependencyFromSpec(raw))
	}
	return
}

// See: ttps://peps.python.org/pep-0503/#normalized-names
var pep503 = regexp.MustCompile(`[-_.]+`)

// NormalizedName returns PEP503-compatible Python Package Index project name.
// As per PEP 426 the only valid characters in a name are the ASCII alphabet,
// ASCII numbers, ., -, and _. The name should be lowercased with all runs of
// the characters ., -, or _ replaced with a single - character.
func (d Distribution) NormalizedName() string {
	return pep503.ReplaceAllString(d.Name, "-")
}

// See: https://peps.python.org/pep-0491/#escaping-and-unicode
var pep491 = regexp.MustCompile(`[^\w\d.]+`)

func (d Distribution) WheelName() string {
	// Each component of the filename is escaped by replacing runs
	// of non-alphanumeric characters with an underscore _
	distName := pep491.ReplaceAllString(d.NormalizedName(), "_")
	return fmt.Sprintf("%s-%s-py3-none-any.whl", distName, d.Version)
}

// ReadDistribution "parses" metadata from setup.py file.
func ReadDistribution(ctx context.Context) (d Distribution, err error) {
	out, err := python.PyInline(ctx, `
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
