package bundletest

import (
	"testing"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/stretchr/testify/assert"
)

func BenchmarkWalkReadOnly(b *testing.B) {
	input := Bundle(b, 10000).Config.View()

	for b.Loop() {
		err := structvar.Walk(input, func(p *structpath.PathNode, v structvar.View) error {
			return nil
		})
		assert.NoError(b, err)
	}
}
