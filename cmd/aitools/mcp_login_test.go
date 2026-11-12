package aitools

import (
	"bytes"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMcpSignInRequiresFullServiceName(t *testing.T) {
	for _, service := range []string{"github", "system.github", "system..github", "a.b.c.d"} {
		t.Run(service, func(t *testing.T) {
			var out mcpSignInOutput
			err := runMcpSignIn(cmdio.MockDiscard(t.Context()), &bytes.Buffer{}, "", service, mcpLogin, &out)
			require.Error(t, err)
			assert.Equal(t, `service must be a full name such as system.ai.github, got "`+service+`"`, err.Error())
			assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))
		})
	}
}
