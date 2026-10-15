package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/settingsv2"
)

// portFwdServerlessSetting lets an administrator disable port forwarding to serverless compute,
// which the driver proxy enforces for the SSH tunnel.
const portFwdServerlessSetting = "port_fwd_serverless_enabled"

func (s *FakeWorkspace) WorkspaceSettingPatch(req Request) Response {
	var setting settingsv2.Setting
	if err := json.Unmarshal(req.Body, &setting); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	// The backend decodes the body as a protobuf message, in which false is the default value, so a
	// body that carries only {"boolean_val": {"value": false}} arrives empty and is rejected.
	if setting.Name == "" && (setting.BooleanVal == nil || !setting.BooleanVal.Value) {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    "Field 'setting' is required and must contain at least one subfield with a non-default value!",
			},
		}
	}

	defer s.LockUnlock()()

	setting.Name = req.Vars["name"]
	// Account-level values are not modelled, so the workspace value is the effective one.
	setting.EffectiveBooleanVal = setting.BooleanVal
	s.WorkspaceSettings[setting.Name] = setting
	return Response{Body: setting}
}

// portForwardingToServerlessDisabled reports whether port_fwd_serverless_enabled was set to false.
// Callers must hold the workspace lock.
func (s *FakeWorkspace) portForwardingToServerlessDisabled() bool {
	setting, ok := s.WorkspaceSettings[portFwdServerlessSetting]
	return ok && setting.EffectiveBooleanVal != nil && !setting.EffectiveBooleanVal.Value
}
