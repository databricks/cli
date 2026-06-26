package fuzz

import (
	"context"
	"encoding/json"
	"errors"
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

// errInvalidConfig marks a generated config that `bundle validate` rejects. The
// caller skips on it: an invalid config can't violate an invariant, so it is not a
// bug. This is the distinction that makes the suite safe to point at a looser
// (e.g. schema-driven) generator, which will produce invalid configs by design.
var errInvalidConfig = errors.New("config did not validate")

// captureJobCreate validates then deploys a bundle containing job via the direct
// engine against an in-process testserver, returning the create request body sent
// to the Jobs API. A validation failure is wrapped as errInvalidConfig. The
// invariant suite asserts properties of the payload; the terraform engine is not
// involved (we assert fundamental properties rather than compare engines).
func captureJobCreate(ctx context.Context, t *testing.T, job *resources.Job) (json.RawMessage, error) {
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
	t.Setenv("DATABRICKS_BUNDLE_ENGINE", "direct")
	t.Chdir(dir)

	// Validate first so an invalid config is reported as errInvalidConfig (caller
	// skips) rather than a deploy failure (caller fails).
	if _, stderr, err := testcli.NewRunner(t, ctx, "bundle", "validate").Run(); err != nil {
		return nil, fmt.Errorf("%w: %v\nstderr:\n%s", errInvalidConfig, err, stderr.String())
	}

	stdout, stderr, err := testcli.NewRunner(t, ctx, "bundle", "deploy").Run()
	if err != nil {
		return nil, fmt.Errorf("bundle deploy failed: %w\nstdout:\n%s\nstderr:\n%s",
			err, stdout.String(), stderr.String())
	}

	body, ok := rec.find("POST", jobsCreatePath)
	if !ok {
		return nil, fmt.Errorf("deploy did not POST %s", jobsCreatePath)
	}
	return body, nil
}

// writeJobBundle writes a minimal databricks.yml for a single job. It emits JSON
// (valid YAML) to reuse the job's own marshaling, which honors ForceSendFields.
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

// fuzzOptInVars opt a run into the invariant suite. FUZZ_SEED(S)/OFFSET also tune
// it (see invariantSeeds); FUZZ_INVARIANTS is a no-tuning switch for `task test-fuzz`.
var fuzzOptInVars = []string{"FUZZ_INVARIANTS", "FUZZ_SEED", "FUZZ_SEEDS", "FUZZ_SEED_OFFSET"}

// requireFuzzOptIn skips unless a FUZZ_* var is set. Each seed runs a real
// in-process deploy, so gating keeps a plain `task test` fast (the single
// un-gated direct smoke test still exercises the harness on every run).
func requireFuzzOptIn(t testing.TB) {
	for _, name := range fuzzOptInVars {
		if os.Getenv(name) != "" {
			return
		}
	}
	t.Skip("invariant fuzz suite is opt-in; run `task test-fuzz` or set FUZZ_SEED=<n> to reproduce a single seed")
}
