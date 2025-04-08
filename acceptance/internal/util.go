package internal

import (
	"regexp"
	"strings"
)

func SubstituteEnv(value string, env []string) (actual, placeholder string) {
	actual = value
	placeholder = value

	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			key, val := parts[0], parts[1]
			re := regexp.MustCompile(`\$` + key + `\b`)
			actual = re.ReplaceAllString(actual, val)
			placeholder = re.ReplaceAllString(placeholder, "["+key+"]")
		}
	}

	return actual, placeholder
}
