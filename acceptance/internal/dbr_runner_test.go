package internal

import (
	"testing"

	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/require"
)

func TestFoo(t *testing.T) {
	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Profile: "azure-ucws-i",
	})
	require.NoError(t, err)

	r := NewDbrRunner(w)
	r.SetDir("testdata/foo")
	r.SetArgs([]string{"bash", "-euo", "pipefail", "somefile"})
	r.AddEnv("HELLO=hi")
	r.AddEnv("WORLD=there")

	err = r.Run()
	require.NoError(t, err)

	// require.Equal(t, "hi\nthere\n", string(r.Output()))
}
