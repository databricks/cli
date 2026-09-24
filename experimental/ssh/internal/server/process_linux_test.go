//go:build linux

package server

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSSHDProcessSetsParentDeathSignal(t *testing.T) {
	cmd := createSSHDProcess(t.Context(), "/tmp/sshd_config")

	require.NotNil(t, cmd.SysProcAttr)
	assert.Equal(t, syscall.SIGKILL, cmd.SysProcAttr.Pdeathsig)
}
