package direct

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestContainsAny(t *testing.T) {
	secrets := []string{"424242", "s3cr3t"}

	assert.True(t, containsAny(dyn.V("name s3cr3t-value"), secrets))
	assert.True(t, containsAny(dyn.V(424242), secrets))
	assert.True(t, containsAny(dyn.V(map[string]dyn.Value{"k": dyn.V([]dyn.Value{dyn.V("x"), dyn.V("a s3cr3t")})}), secrets))

	assert.False(t, containsAny(dyn.V("job-a"), secrets))
	assert.False(t, containsAny(dyn.V(42), secrets))
	assert.False(t, containsAny(dyn.V("s3cr3t"), nil))
}
