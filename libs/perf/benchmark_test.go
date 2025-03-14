package perf

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"
	"github.com/stretchr/testify/require"
)

func TestFoo(t *testing.T) {
	s := StartServer()
	defer s.Close()

	b := Body{Args: []string{"bundle", "deploy"}}
	encoded, err := json.Marshal(b)
	require.NoError(t, err)

	_, err = http.Post(s.URL, "application/json", bytes.NewReader(encoded))
	require.NoError(t, err)
}

func BenchmarkBundleDeploy(t *testing.T) {
	os.Chdir("/Users/shreyas.goenka/projects/bundle-playground")

	cli := cmd.New(context.Background())
	cli.SetArgs([]string{"bundle", "deploy"})
	err := root.Execute(cli.Context(), cli)
	require.NoError(t, err)
}
