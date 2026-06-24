package fuzz

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/testserver"
)

const (
	// bundleResourceKey is the map key the generated job is registered under.
	bundleResourceKey = "fuzz_job"
	fakeToken         = "testtoken"
)

// captureJobCreate deploys a bundle containing job through the given engine
// ("direct" or "terraform") and returns the create request body sent to the
// Jobs API.
//
// Both engines run the full `bundle deploy` pipeline against an in-process
// testserver, so the only difference between two captures with different engines
// is the engine itself. That is what makes the resulting payloads directly
// comparable: shared mutators (deployment metadata, presets, ...) are applied
// identically on both sides and cancel out in the diff.
//
// The terraform engine additionally requires DATABRICKS_TF_EXEC_PATH and
// DATABRICKS_TF_CLI_CONFIG_FILE to point at a provisioned terraform binary and
// provider mirror; see requireTerraform.
func captureJobCreate(ctx context.Context, t *testing.T, job *resources.Job, engine string) (json.RawMessage, error) {
	rec := &recorder{}
	server := testserver.New(t)
	server.RequestCallback = rec.callback
	testserver.AddDefaultHandlers(server)

	dir := t.TempDir()
	if err := writeJobBundle(dir, server.URL, job); err != nil {
		return nil, err
	}

	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", fakeToken)
	t.Setenv("DATABRICKS_BUNDLE_ENGINE", engine)
	t.Chdir(dir)

	stdout, stderr, err := testcli.NewRunner(t, ctx, "bundle", "deploy").Run()
	if err != nil {
		return nil, fmt.Errorf("bundle deploy (engine=%s) failed: %w\nstdout:\n%s\nstderr:\n%s",
			engine, err, stdout.String(), stderr.String())
	}

	body, ok := rec.find("POST", jobsCreatePath)
	if !ok {
		return nil, fmt.Errorf("engine=%s did not POST %s during deploy", engine, jobsCreatePath)
	}
	return body, nil
}

// compareJobEngines deploys job under both engines and returns the create-payload
// differences that are not covered by DefaultIgnorePaths. An empty result means
// the engines produced equivalent create payloads.
func compareJobEngines(ctx context.Context, t *testing.T, job *resources.Job) ([]Difference, error) {
	direct, err := captureJobCreate(ctx, t, job, "direct")
	if err != nil {
		return nil, fmt.Errorf("capturing direct payload: %w", err)
	}
	terraform, err := captureJobCreate(ctx, t, job, "terraform")
	if err != nil {
		return nil, fmt.Errorf("capturing terraform payload: %w", err)
	}
	return DiffPayloads(direct, terraform, DefaultIgnorePaths)
}

// writeJobBundle writes a minimal databricks.yml describing a single job. The
// document is emitted as JSON, which is valid YAML, so we can reuse the job's
// own JSON marshaling (which honors ForceSendFields) without a YAML dependency.
func writeJobBundle(dir, host string, job *resources.Job) error {
	jobJSON, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshaling job: %w", err)
	}

	var jobMap map[string]any
	if err := json.Unmarshal(jobJSON, &jobMap); err != nil {
		return fmt.Errorf("unmarshaling job: %w", err)
	}

	doc := map[string]any{
		"bundle":    map[string]any{"name": "fuzz"},
		"workspace": map[string]any{"host": host},
		"resources": map[string]any{
			"jobs": map[string]any{bundleResourceKey: jobMap},
		},
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling bundle: %w", err)
	}

	return os.WriteFile(filepath.Join(dir, "databricks.yml"), data, 0o600)
}

// fuzzOptInVars are the environment variables that opt a run into the
// terraform-backed parity suite. FUZZ_SEED / FUZZ_SEEDS / FUZZ_SEED_OFFSET double
// as the tuning knobs (see paritySeeds), so setting any of them implies opt-in;
// FUZZ_PARITY is a no-tuning switch used by `task test-fuzz`.
var fuzzOptInVars = []string{"FUZZ_PARITY", "FUZZ_SEED", "FUZZ_SEEDS", "FUZZ_SEED_OFFSET"}

// requireFuzzOptIn skips unless the run explicitly opted into the terraform
// parity suite. Gating on an env var rather than on the presence of build/ keeps
// a leftover terraform install (from a prior `task test-fuzz` or acceptance run)
// from silently turning a plain `task test` into dozens of real deploys.
func requireFuzzOptIn(t testing.TB) {
	for _, name := range fuzzOptInVars {
		if os.Getenv(name) != "" {
			return
		}
	}
	t.Skip("terraform parity suite is opt-in; run `task test-fuzz` or set FUZZ_SEED=<n> to reproduce a single seed")
}

// requireTerraform opts in via requireFuzzOptIn, then points the terraform engine
// at the binary and provider mirror provisioned by acceptance/install_terraform.py
// into <repo>/build, skipping when they are absent so the suite still skips
// cleanly where terraform is not set up.
func requireTerraform(t testing.TB) {
	requireFuzzOptIn(t)

	buildDir := filepath.Join(repoRoot(t), "build")
	execPath := filepath.Join(buildDir, "terraform")
	cfgFile := filepath.Join(buildDir, ".terraformrc")

	// install_terraform.py provisions all three together; a partial build/ (e.g.
	// the binary without the provider mirror or .terraformrc) would otherwise fail
	// mid-deploy with a confusing error instead of skipping cleanly.
	tfpluginsDir := filepath.Join(buildDir, "tfplugins")
	for _, p := range []string{execPath, cfgFile, tfpluginsDir} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("terraform not fully provisioned (%s); run: python3 acceptance/install_terraform.py --targetdir build", p)
		}
	}

	t.Setenv("DATABRICKS_TF_EXEC_PATH", execPath)
	t.Setenv("DATABRICKS_TF_CLI_CONFIG_FILE", cfgFile)
	t.Setenv("TF_CLI_CONFIG_FILE", cfgFile)
	// Terraform phones home to checkpoint-api.hashicorp.com otherwise; disable it
	// so the testserver/network isn't hit. See acceptance_test.go.
	t.Setenv("CHECKPOINT_DISABLE", "1")
}

// repoRoot returns the repository root by walking up from the current directory.
func repoRoot(t testing.TB) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %s", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repo root (go.mod not found)")
		}
		dir = parent
	}
}
