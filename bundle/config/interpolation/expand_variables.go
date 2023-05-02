package interpolation

import (
	"fmt"
	"strings"
)

const VariableReferencePrefix = "var."

func isVariableReference(s string) bool {
	if !strings.HasPrefix(s, VariableReferencePrefix) || strings.Count(s, ".") != 1 {
		return false
	}
	name := strings.TrimPrefix(s, VariableReferencePrefix)
	return len(name) > 0
}

// Expands variable references of the form `var.foo` to `variables.foo.value`
// Errors out if input string is not in the correct format
func expandVariable(s string) (string, error) {
	if !isVariableReference(s) {
		return "", fmt.Errorf("%s is not a valid variable reference", s)
	}
	name := strings.TrimPrefix(s, VariableReferencePrefix)
	return strings.Join([]string{"variables", name, "value"}, "."), nil
}
