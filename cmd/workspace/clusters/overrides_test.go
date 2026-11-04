package clusters

import (
	"bytes"
	"errors"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartOverride_SwallowsInvalidStateError(t *testing.T) {
	startReq := &compute.StartCluster{ClusterId: "1234-000000-abcdefg"}
	cmd := &cobra.Command{}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return &apierr.APIError{ErrorCode: "INVALID_STATE", Message: "Cluster is in unexpected state Running."}
	}

	startOverride(cmd, startReq)

	var out bytes.Buffer
	cmd.SetOut(&out)
	err := cmd.RunE(cmd, nil)

	require.NoError(t, err)
	assert.Contains(t, out.String(), startReq.ClusterId)
}

func TestStartOverride_PropagatesOtherErrors(t *testing.T) {
	startReq := &compute.StartCluster{ClusterId: "1234-000000-abcdefg"}
	cmd := &cobra.Command{}
	wantErr := errors.New("boom")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return wantErr
	}

	startOverride(cmd, startReq)

	err := cmd.RunE(cmd, nil)

	assert.Equal(t, wantErr, err)
}
