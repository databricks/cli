package internal

import (
	"strings"
)

// Substitute env variables like $VAR in value.
// value is a string with variable references like $VAR
// env is a set of variables in golang format, like VAR=hello
// Example: value="$CLI", env={"CLI=/bin/true"}, result: "/bin/true"
func SubstituteEnv(value string, env []string) string {
	envMap := make(map[string]string)
	
	// Parse environment variables into a map
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	
	// Replace $VAR references with their values
	result := value
	for k, v := range envMap {
		result = strings.ReplaceAll(result, "$"+k, v)
	}
	
	return result
}
