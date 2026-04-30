package internal

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
)

const MaterializedConfigFile = "out.test.toml"

// GenerateMaterializedConfig creates a TOML representation of the configuration fields
// that determine where and how a test is executed.
func GenerateMaterializedConfig(config *TestConfig) string {
	var buf strings.Builder

	writeBool(&buf, "Local", config.Local)
	writeBool(&buf, "Cloud", config.Cloud)
	writeBool(&buf, "CloudSlow", config.CloudSlow)
	writeBool(&buf, "RequiresUnityCatalog", config.RequiresUnityCatalog)
	writeBool(&buf, "RequiresCluster", config.RequiresCluster)
	writeBool(&buf, "RequiresWarehouse", config.RequiresWarehouse)
	writeBool(&buf, "RunsOnDbr", config.RunsOnDbr)
	if config.Phase != 0 {
		fmt.Fprintf(&buf, "Phase = %d\n", config.Phase)
	}

	for _, k := range slices.Sorted(maps.Keys(config.GOOS)) {
		fmt.Fprintf(&buf, "GOOS.%s = %v\n", k, config.GOOS[k])
	}
	for _, k := range slices.Sorted(maps.Keys(config.CloudEnvs)) {
		fmt.Fprintf(&buf, "CloudEnvs.%s = %v\n", k, config.CloudEnvs[k])
	}
	for _, k := range slices.Sorted(maps.Keys(config.EnvMatrix)) {
		writeTomlStringArray(&buf, "EnvMatrix."+k, config.EnvMatrix[k])
	}

	return buf.String()
}

func writeBool(buf *strings.Builder, key string, v *bool) {
	if v != nil {
		fmt.Fprintf(buf, "%s = %v\n", key, *v)
	}
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
