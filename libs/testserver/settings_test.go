package testserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func patchSetting(workspace *FakeWorkspace, body string) Response {
	return workspace.WorkspaceSettingPatch(Request{
		Body: []byte(body),
		Vars: map[string]string{"name": portFwdServerlessSetting},
	})
}

func TestWorkspaceSettingPatchRejectsBodyWithOnlyDefaults(t *testing.T) {
	workspace := NewFakeWorkspace("http://test.test", "token")

	response := patchSetting(workspace, `{"boolean_val": {"value": false}}`)

	assert.Equal(t, 400, response.StatusCode)
	assert.False(t, workspace.portForwardingToServerlessDisabled())
}

func TestWorkspaceSettingPatchDisablesPortForwarding(t *testing.T) {
	workspace := NewFakeWorkspace("http://test.test", "token")

	response := patchSetting(workspace, `{"name": "port_fwd_serverless_enabled", "boolean_val": {"value": false}}`)

	assert.Equal(t, 0, response.StatusCode)
	assert.True(t, workspace.portForwardingToServerlessDisabled())
}
