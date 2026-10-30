package structpath_test

import (
	"testing"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/stretchr/testify/assert"
)

func TestPureReferenceMatchesStructvar(t *testing.T) {
	for _, s := range []string{
		"${var.foo}",
		"${resources.jobs.foo.tasks[1].env.key}",
		"${var.résumé_2}",
		"${_private.x}",
		"${var.foo[0][1]}",
		"${var.jobs['my_job']}",
		"${var.-foo}",
		"${var.foo-}",
		"${var.foo}${var.bar}",
		"prefix_${var.field}",
		"$${var.escaped}",
		"${}",
		"plain_string",
	} {
		_, ok := structpath.PureReferenceToPath(s)
		assert.Equal(t, structvar.IsPureVariableReference(s), ok, s)
	}
}
