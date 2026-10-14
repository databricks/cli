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
