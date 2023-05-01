package interpolation

import (
	"fmt"
	"strings"
)

const VariableReferencePrefix = "var."

func isVariableReference(s string) bool {
	return strings.HasPrefix(s, VariableReferencePrefix) && strings.Count(s, ".") == 1
}

// Expands variable references of the form `var.foo` to `variables.foo.value`
// Errors out if input string is not in the correct format
// TODO: add tests for this
func expandVariable(s string) (string, error) {
	if !isVariableReference(s) {
		return "", fmt.Errorf("%s is not a variable reference", s)
	}
	name := strings.TrimPrefix(s, VariableReferencePrefix)
	return strings.Join([]string{"variables", name, "value"}, "."), nil
}
