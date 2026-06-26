package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/databricks/cli/experimental/authdoctor"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitScopes(t *testing.T) {
	assert.Nil(t, splitScopes(""))
	assert.Equal(t, []string{"all-apis"}, splitScopes("all-apis"))
	assert.Equal(t, []string{"jobs", "sql"}, splitScopes(" jobs , sql "))
	assert.Equal(t, []string{"jobs"}, splitScopes("jobs,,"))
}

func TestEffectiveAuthType(t *testing.T) {
	assert.Equal(t, "pat", effectiveAuthType(&config.Config{AuthType: "pat"}))
	assert.Equal(t, "pat", effectiveAuthType(&config.Config{Token: "dapi123"}))
	assert.Equal(t, "oauth-m2m", effectiveAuthType(&config.Config{ClientID: "a", ClientSecret: "b"}))
	assert.Equal(t, authdoctor.AuthTypeDatabricksCLI, effectiveAuthType(&config.Config{}))
	// An explicit auth_type wins over inferred credentials.
	assert.Equal(t, "azure-cli", effectiveAuthType(&config.Config{AuthType: "azure-cli", Token: "dapi"}))
	// A non-U2M credential signal without an explicit auth_type must NOT be
	// inferred as databricks-cli (would trigger spurious U2M token checks).
	assert.Equal(t, authTypeUnknown, effectiveAuthType(&config.Config{AzureResourceID: "/subscriptions/x"}))
	assert.Equal(t, authTypeUnknown, effectiveAuthType(&config.Config{Username: "u", Password: "p"}))
	assert.Equal(t, authTypeUnknown, effectiveAuthType(&config.Config{MetadataServiceURL: "http://localhost"}))
	assert.False(t, authdoctor.IsU2M(authTypeUnknown))
}

func TestClassifyHostType(t *testing.T) {
	assert.Equal(t, authdoctor.HostTypeAccount, classifyHostType("https://accounts.cloud.databricks.com"))
	assert.Equal(t, authdoctor.HostTypeAccount, classifyHostType("accounts.cloud.databricks.com"))
	assert.Equal(t, authdoctor.HostTypeWorkspace, classifyHostType("https://myws.cloud.databricks.com"))
	assert.Equal(t, authdoctor.HostTypeWorkspace, classifyHostType(""))
}

func TestResolveProfileFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("profile", "", "")
	require.NoError(t, cmd.Flags().Set("profile", "myprofile"))
	cmd.SetContext(t.Context())

	name, src := resolveProfile(cmd.Context(), cmd)
	assert.Equal(t, "myprofile", name)
	assert.Equal(t, authdoctor.SourceFlag, src)
}

func TestResolveProfileEnv(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("profile", "", "")
	ctx := env.Set(t.Context(), "DATABRICKS_CONFIG_PROFILE", "envprofile")
	cmd.SetContext(ctx)

	name, src := resolveProfile(ctx, cmd)
	assert.Equal(t, "envprofile", name)
	assert.Equal(t, authdoctor.SourceEnvProfile, src)
}

func newRenderCmd(t *testing.T, output flags.Output) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{}
	out := output
	cmd.Flags().Var(&out, "output", "")
	cmd.SetContext(t.Context())
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	return cmd, buf
}

var sampleReports = []authdoctor.Report{{
	Profile:  "DEFAULT",
	Host:     "https://myworkspace.cloud.databricks.test",
	AuthType: "databricks-cli",
	Overall:  authdoctor.LevelWarn,
	Findings: []authdoctor.Finding{
		{Check: "host", Level: authdoctor.LevelOK, Title: "Host is a workspace URL"},
		{Check: "scopes", Level: authdoctor.LevelWarn, Title: "Profile is downscoped", Detail: "limited to jobs", Fix: "databricks auth login --scopes 'all-apis'"},
	},
}}

func TestRenderReportsJSON(t *testing.T) {
	cmd, buf := newRenderCmd(t, flags.OutputJSON)
	require.NoError(t, renderReports(cmd.Context(), cmd, sampleReports))

	var got []authdoctor.Report
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got, 1)
	assert.Equal(t, "DEFAULT", got[0].Profile)
	assert.Equal(t, authdoctor.LevelWarn, got[0].Overall)
	assert.Len(t, got[0].Findings, 2)
}

func TestRenderReportsText(t *testing.T) {
	cmd, buf := newRenderCmd(t, flags.OutputText)
	require.NoError(t, renderReports(cmd.Context(), cmd, sampleReports))

	s := buf.String()
	assert.Contains(t, s, "Profile \"DEFAULT\"")
	assert.Contains(t, s, "Host is a workspace URL")
	assert.Contains(t, s, "Profile is downscoped")
	assert.Contains(t, s, "Fix: databricks auth login --scopes 'all-apis'")
	assert.Contains(t, s, "Overall:")
}

func TestRenderReportsTextEmpty(t *testing.T) {
	cmd, buf := newRenderCmd(t, flags.OutputText)
	require.NoError(t, renderReports(cmd.Context(), cmd, nil))
	assert.Contains(t, buf.String(), "No profiles found.")
}
