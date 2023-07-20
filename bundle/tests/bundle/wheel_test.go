package bundle

import (
	"context"
	"os"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/stretchr/testify/require"
)

func TestBundlePythonWheelBuild(t *testing.T) {
	b, err := bundle.Load("./python_wheel")
	require.NoError(t, err)

	m := phases.Build()
	err = m.Apply(context.Background(), b)
	require.NoError(t, err)

	_, err = os.Stat("./python_wheel/my_test_code/dist/my_test_code-0.0.1-py2-none-any.whl")
	require.NoError(t, err)
}
