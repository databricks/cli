package testserver

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/settingsv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestWorkspaceSettingGetDefaultsPortForwardingToEnabled(t *testing.T) {
	workspace := NewFakeWorkspace("http://test.test", "token")

	response := workspace.WorkspaceSettingGet(Request{Vars: map[string]string{"name": portFwdServerlessSetting}})

	assert.Equal(t, 0, response.StatusCode)
	setting, ok := response.Body.(settingsv2.Setting)
	require.True(t, ok)
	require.NotNil(t, setting.EffectiveBooleanVal)
	assert.True(t, setting.EffectiveBooleanVal.Value)
}

func TestWorkspaceSettingGetReturnsPatchedValue(t *testing.T) {
	workspace := NewFakeWorkspace("http://test.test", "token")
	patchSetting(workspace, `{"name": "port_fwd_serverless_enabled", "boolean_val": {"value": false}}`)

	response := workspace.WorkspaceSettingGet(Request{Vars: map[string]string{"name": portFwdServerlessSetting}})

	setting, ok := response.Body.(settingsv2.Setting)
	require.True(t, ok)
	require.NotNil(t, setting.EffectiveBooleanVal)
	assert.False(t, setting.EffectiveBooleanVal.Value)
}

func TestWorkspaceSettingGetUnknownSetting(t *testing.T) {
	workspace := NewFakeWorkspace("http://test.test", "token")

	response := workspace.WorkspaceSettingGet(Request{Vars: map[string]string{"name": "no_such_setting"}})

	assert.Equal(t, 404, response.StatusCode)
}
