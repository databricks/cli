package internal

import (
	"regexp"
	"strings"
)

// isAlphaNumeric returns true if the rune is a letter or digit
func isAlphaNumeric(r byte) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// Substitute env variables like $VAR in value.
// value is a string with variable references like $VAR
// env is a set of variables in golang format, like VAR=hello
// Example: value="$CLI", env={"CLI=/bin/true"}, result: "/bin/true"
func SubstituteEnv(value string, env []string) string {
	result := value
	
	// Process environment variables in the order they appear in the input slice
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			key, val := parts[0], parts[1]
			// Create a regexp that matches $VAR but not $VARNAME (where NAME is alphanumeric)
			re := regexp.MustCompile(`\$` + key + `\b`)
			result = re.ReplaceAllString(result, val)
		}
	}

	return result
}
