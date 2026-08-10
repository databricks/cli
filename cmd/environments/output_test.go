package environments

import (
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	libslocalenv "github.com/databricks/cli/libs/localenv"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// renderText runs renderResult in text mode and returns what was written to
// stderr (where cmdio.LogString writes). The command is wired with a proper
// *flags.Output persistent flag so root.OutputType does not panic.
func renderText(t *testing.T, res *libslocalenv.Result, pipelineErr error) string {
	t.Helper()
	ctx, buf := cmdio.NewTestContextWithStderr(t.Context())
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	// Register the output flag as *flags.Output (text default) so root.OutputType
	// finds a *flags.Output value and returns OutputText without panicking.
	output := flags.OutputText
	cmd.PersistentFlags().VarP(&output, "output", "o", "output type: text or json")
	_ = renderResult(ctx, cmd, res, pipelineErr)
	return buf.String()
}

func TestRenderSuccessSummary(t *testing.T) {
	res := libslocalenv.NewResult()
	res.OK = true
	res.Mode = libslocalenv.ModeDefault.String()
	res.Compute = &libslocalenv.ComputeInfo{Source: "serverless", ServerlessVersion: "v4", EnvKey: "serverless/serverless-v4"}
	res.Resolved = &libslocalenv.ResolvedInfo{PythonVersion: "3.12", DBConnectVersion: "16.1.0"}
	res.VenvPath = ".venv"
	res.BackupPath = "pyproject.toml.bak"

	out := renderText(t, res, nil)

	assert.Contains(t, out, "Local environment ready")
	assert.Contains(t, out, "serverless 4")
	assert.Contains(t, out, "3.12")
	assert.Contains(t, out, "16.1.0")
	assert.Contains(t, out, ".venv")
	assert.Contains(t, out, "pyproject.toml.bak")
	assert.Contains(t, out, "Next steps")
	// The raw phase log must NOT appear on success anymore.
	assert.NotContains(t, out, "preflight  ok")
}

func TestRenderSuccessConstraintsOnlyOmitsDBConnect(t *testing.T) {
	res := libslocalenv.NewResult()
	res.OK = true
	res.Mode = libslocalenv.ModeConstraintsOnly.String()
	res.Compute = &libslocalenv.ComputeInfo{Source: "serverless", ServerlessVersion: "v4", EnvKey: "serverless/serverless-v4"}
	res.Resolved = &libslocalenv.ResolvedInfo{PythonVersion: "3.12"} // no DBConnectVersion
	res.VenvPath = ".venv"

	out := renderText(t, res, nil)
	// renderSuccess omits the row when DBConnectVersion is empty — which is how constraints-only mode leaves it.
	assert.NotContains(t, out, "databricks-connect")
}

func TestRenderFailure(t *testing.T) {
	res := libslocalenv.NewResult()
	res.Phases = []libslocalenv.PhaseStatus{}
	res.Error = &libslocalenv.PipelineError{
		Code:         libslocalenv.ErrFetch,
		FailurePhase: libslocalenv.PhaseFetch,
		Msg:          "constraint repo unreachable",
	}
	perr := res.Error

	out := renderText(t, res, perr)

	assert.Contains(t, out, "Setup failed")
	assert.Contains(t, out, "fetching constraints")
	assert.Contains(t, out, "constraint repo unreachable")
	assert.Contains(t, out, "--debug")
}

func TestRenderCanceled(t *testing.T) {
	res := libslocalenv.NewResult()
	res.Error = &libslocalenv.PipelineError{Code: libslocalenv.ErrCanceled, FailurePhase: libslocalenv.PhaseProvision, Msg: "interrupted"}
	out := renderText(t, res, res.Error)
	assert.Contains(t, out, "canceled")
}

func TestRenderDryRun(t *testing.T) {
	res := libslocalenv.NewResult()
	res.OK = true
	res.DryRun = true
	res.Plan = &libslocalenv.Plan{WouldWrite: "/tmp/pyproject.toml", ChangedRegions: []string{"requires-python"}}
	out := renderText(t, res, nil)
	assert.Contains(t, out, "requires-python")
	assert.Contains(t, out, "No files were modified")
}

func TestActivateHint(t *testing.T) {
	hint := activateHint(".venv")
	if runtime.GOOS == "windows" {
		// Windows: uv lays the venv out under Scripts\, and "source" is not a
		// cmd/PowerShell builtin, so the hint must not suggest it.
		assert.Equal(t, `.venv\Scripts\activate`, hint)
		assert.NotContains(t, hint, "source ")
		assert.NotContains(t, hint, "/bin/")
	} else {
		assert.Equal(t, "source .venv/bin/activate", hint)
	}
}
