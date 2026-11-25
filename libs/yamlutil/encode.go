// Package yamlutil holds the YAML helpers shared across the CLI.
package yamlutil

import (
	"io"

	"go.yaml.in/yaml/v4"
)

// Encode writes v to w as a YAML document indented with two spaces.
//
// yaml.NewEncoder wraps lines at 80 columns in go-yaml v4.0.0 and has no setter
// for the width (yaml/go-yaml#525), so the options are spelled out to keep long
// lines intact.
func Encode(w io.Writer, v any) error {
	dumper, err := yaml.NewDumper(w, yaml.WithV3Defaults(), yaml.WithIndent(2), yaml.WithLineWidth(-1))
	if err != nil {
		return err
	}
	if err := dumper.Dump(v); err != nil {
		return err
	}
	return dumper.Close()
}
