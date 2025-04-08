package internal

import (
	"strings"
	"unicode"
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
	envMap := make(map[string]string)
	
	// Parse environment variables into a map
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	
	// Replace $VAR references with their values
	// Use a more precise replacement to handle overlapping variable names
	result := value
	for k, v := range envMap {
		// Match $VAR followed by space, end of string, or non-alphanumeric character
		parts := strings.Split(result, "$"+k)
		if len(parts) <= 1 {
			continue
		}
		
		var newResult strings.Builder
		newResult.WriteString(parts[0])
		
		for i := 1; i < len(parts); i++ {
			// Check if this is a complete variable name (not a prefix of another)
			if len(parts[i]) == 0 || !isAlphaNumeric(parts[i][0]) {
				newResult.WriteString(v)
			} else {
				newResult.WriteString("$" + k)
			}
			newResult.WriteString(parts[i])
		}
		
		result = newResult.String()
	}
	
	return result
}
