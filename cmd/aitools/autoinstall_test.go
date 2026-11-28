package aitools

import (
	"testing"

	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/skills"
	"github.com/stretchr/testify/assert"
)

// enable says when DATABRICKS_AGENT_FEATURES_DISABLED still turns auto-install
// off, as it does for DATABRICKS_AITOOLS_AUTO_INSTALL; unset, it says nothing
// about it.
func TestPrintAutoInstallSettingAgentFeaturesDisabled(t *testing.T) {
	const warning = "DATABRICKS_AGENT_FEATURES_DISABLED still disables it in this environment; unset it to let it run."

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.Set(ctx, skills.AgentFeaturesDisabledEnv, "")
	printAutoInstallSetting(ctx, true)
	assert.NotContains(t, stderr.String(), skills.AgentFeaturesDisabledEnv)

	ctx, stderr = cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.Set(ctx, skills.AgentFeaturesDisabledEnv, "1")
	printAutoInstallSetting(ctx, true)
	assert.Contains(t, stderr.String(), "Automatic installation of Databricks AI tools is enabled.")
	assert.Contains(t, stderr.String(), warning)
}

// With both variables set, one warning names both, so unsetting only one of
// them is not taken for a fix.
func TestPrintAutoInstallSettingBothOverrides(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.Set(ctx, autoinstall.EnvVar, "false")
	ctx = env.Set(ctx, skills.AgentFeaturesDisabledEnv, "1")
	printAutoInstallSetting(ctx, true)
	assert.Contains(t, stderr.String(), autoinstall.EnvVar+" and "+skills.AgentFeaturesDisabledEnv+
		" still disable it in this environment; unset both to let it run.")
	assert.NotContains(t, stderr.String(), "unset it to let it run")
}
