//go:build !linux

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateSSHDProcessLeavesSysProcAttrUnset(t *testing.T) {
	cmd := createSSHDProcess(t.Context(), "/tmp/sshd_config")

	assert.Nil(t, cmd.SysProcAttr)
}
