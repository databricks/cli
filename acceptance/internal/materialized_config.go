package internal

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
)

const MaterializedConfigFile = "out.test.toml"

type MaterializedConfig struct {
	GOOS                 map[string]bool     `toml:"GOOS,omitempty"`
	CloudEnvs            map[string]bool     `toml:"CloudEnvs,omitempty"`
	Local                *bool               `toml:"Local,omitempty"`
	Cloud                *bool               `toml:"Cloud,omitempty"`
	CloudSlow            *bool               `toml:"CloudSlow,omitempty"`
	RequiresUnityCatalog *bool               `toml:"RequiresUnityCatalog,omitempty"`
	RequiresCluster      *bool               `toml:"RequiresCluster,omitempty"`
	RequiresWarehouse    *bool               `toml:"RequiresWarehouse,omitempty"`
	RunsOnDbr            *bool               `toml:"RunsOnDbr,omitempty"`
	Phase                *int                `toml:"Phase,omitempty"`
	EnvMatrix            map[string][]string `toml:"EnvMatrix,omitempty"`
}

// GenerateMaterializedConfig creates a TOML representation of the configuration fields
// that determine where and how a test is executed.
func GenerateMaterializedConfig(config TestConfig) (string, error) {
	var phase *int
	if config.Phase != 0 {
		phase = &config.Phase
	}

	materialized := MaterializedConfig{
		GOOS:                 config.GOOS,
		CloudEnvs:            config.CloudEnvs,
		Local:                config.Local,
		Cloud:                config.Cloud,
		CloudSlow:            config.CloudSlow,
		RequiresUnityCatalog: config.RequiresUnityCatalog,
		RequiresCluster:      config.RequiresCluster,
		RequiresWarehouse:    config.RequiresWarehouse,
		RunsOnDbr:            config.RunsOnDbr,
		Phase:                phase,
		EnvMatrix:            config.EnvMatrix,
	}

	return encodeMaterializedConfig(materialized), nil
}

func encodeMaterializedConfig(c MaterializedConfig) string {
	var buf strings.Builder

	writeBool := func(key string, v *bool) {
		if v != nil {
			fmt.Fprintf(&buf, "%s = %v\n", key, *v)
		}
	}
	writeBool("Local", c.Local)
	writeBool("Cloud", c.Cloud)
	writeBool("CloudSlow", c.CloudSlow)
	writeBool("RequiresUnityCatalog", c.RequiresUnityCatalog)
	writeBool("RequiresCluster", c.RequiresCluster)
	writeBool("RequiresWarehouse", c.RequiresWarehouse)
	writeBool("RunsOnDbr", c.RunsOnDbr)
	if c.Phase != nil {
		fmt.Fprintf(&buf, "Phase = %d\n", *c.Phase)
	}

	for _, k := range slices.Sorted(maps.Keys(c.GOOS)) {
		fmt.Fprintf(&buf, "GOOS.%s = %v\n", k, c.GOOS[k])
	}
	for _, k := range slices.Sorted(maps.Keys(c.CloudEnvs)) {
		fmt.Fprintf(&buf, "CloudEnvs.%s = %v\n", k, c.CloudEnvs[k])
	}
	for _, k := range slices.Sorted(maps.Keys(c.EnvMatrix)) {
		writeTomlStringArray(&buf, "EnvMatrix."+k, c.EnvMatrix[k])
	}

	return buf.String()
}

// writeTomlStringArray writes a TOML string array. Arrays with more than 3 elements
// use one element per line for readability.
func writeTomlStringArray(buf *strings.Builder, key string, vals []string) {
	if len(vals) > 3 {
		fmt.Fprintf(buf, "%s = [\n", key)
		for i, v := range vals {
			if i < len(vals)-1 {
				fmt.Fprintf(buf, "  %s,\n", tomlQuote(v))
			} else {
				fmt.Fprintf(buf, "  %s\n", tomlQuote(v))
			}
		}
		buf.WriteString("]\n")
		return
	}
	fmt.Fprintf(buf, "%s = [", key)
	for i, v := range vals {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(tomlQuote(v))
	}
	buf.WriteString("]\n")
}

// tomlQuote returns a TOML basic string literal for s using JSON encoding,
// whose escape sequences (\", \\, \n, \r, \t, \uXXXX) are all valid in TOML.
func tomlQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
