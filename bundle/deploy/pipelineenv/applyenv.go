// Package pipelineenv calls pipelines.ApplyEnvironment after bundle deploy
// to refresh dev-mode pipelines when pyproject.toml or a wheel artifact changes.
package pipelineenv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

const hashFileName = "pyproject_env_hash"

type applyEnv struct{}

func (a *applyEnv) Name() string { return "pipelineenv.ApplyEnvironment" }

func ApplyEnvironment() bundle.Mutator { return &applyEnv{} }

func (a *applyEnv) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	affected := pipelinesNeedingEnvApply(b)
	if len(affected) == 0 {
		return nil
	}

	newHash, err := pyprojectHash(b)
	if err != nil {
		log.Debugf(ctx, "pipelineenv: skipping; could not hash pyproject.toml: %v", err)
		return nil
	}
	prevHash := readPrevHash(ctx, b)
	hashChanged := newHash != "" && newHash != prevHash

	if !hashChanged && !hasWheelArtifact(b.Config.Artifacts) {
		return nil
	}

	names := make([]string, 0, len(affected))
	for _, p := range affected {
		names = append(names, p.Name)
	}
	cmdio.LogString(ctx, fmt.Sprintf("Re-applying environment for pipeline(s) %s (using databricks pipelines apply-environment)", strings.Join(names, ", ")))

	w := b.WorkspaceClient(ctx)
	allOK := true
	for _, p := range affected {
		_, err := w.Pipelines.ApplyEnvironment(ctx, pipelines.ApplyEnvironmentRequest{PipelineId: p.ID})
		if err == nil {
			continue
		}
		// Pipelines apply-environment requires the pipeline cluster to be running.
		// In dev mode the cluster auto-stops between updates; when it has stopped
		// the API returns 404 with "Pipeline compute … is not found". The next
		// update will install a fresh env naturally, so treat this as a no-op.
		if isComputeNotRunning(err) {
			log.Debugf(ctx, "pipelineenv: %s compute not running; next update will refresh env", p.Name)
			continue
		}
		log.Warnf(ctx, "pipelineenv: failed for pipeline %s: %v", p.Name, err)
		allOK = false
	}

	// Persist the hash only if every call succeeded (or soft-succeeded), so
	// transient failures retry on the next deploy.
	if allOK && hashChanged {
		if err := writePrevHash(ctx, b, newHash); err != nil {
			log.Debugf(ctx, "pipelineenv: failed to persist hash: %v", err)
		}
	}
	return nil
}

// pipelinesNeedingEnvApply returns the dev-mode classic-compute pipelines that
// could be affected by env cache. Skipped:
//   - missing/not-yet-deployed (no ID)
//   - production mode (env always reinstalls on every update)
//   - serverless (each update gets a fresh cluster; env refreshes naturally)
func pipelinesNeedingEnvApply(b *bundle.Bundle) []*resources.Pipeline {
	keys := make([]string, 0, len(b.Config.Resources.Pipelines))
	for k := range b.Config.Resources.Pipelines {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	out := make([]*resources.Pipeline, 0, len(keys))
	for _, k := range keys {
		p := b.Config.Resources.Pipelines[k]
		if p == nil || p.ID == "" || !p.Development || p.Serverless {
			continue
		}
		out = append(out, p)
	}
	return out
}

func hasWheelArtifact(artifacts config.Artifacts) bool {
	for _, art := range artifacts {
		if art != nil && art.Type == config.ArtifactPythonWheel {
			return true
		}
	}
	return false
}

// pyprojectHash returns the sha256 of the bundle's pyproject.toml, or "" if
// the file doesn't exist (bundles without a Python project are valid).
func pyprojectHash(b *bundle.Bundle) (string, error) {
	f, err := os.Open(filepath.Join(b.BundleRootPath, "pyproject.toml"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func readPrevHash(ctx context.Context, b *bundle.Bundle) string {
	cacheDir, err := b.LocalStateDir(ctx)
	if err != nil {
		return ""
	}
	data, _ := os.ReadFile(filepath.Join(cacheDir, hashFileName))
	return string(data)
}

func writePrevHash(ctx context.Context, b *bundle.Bundle, hash string) error {
	cacheDir, err := b.LocalStateDir(ctx)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cacheDir, hashFileName), []byte(hash), 0o600)
}

func isComputeNotRunning(err error) bool {
	var apiErr *apierr.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.StatusCode != http.StatusNotFound {
		return false
	}
	// Distinguish "compute is idle" from other 404s (e.g. wrong pipeline id) by
	// checking the message preamble. The SDK exposes no sentinel for this case.
	return strings.Contains(apiErr.Message, "Pipeline compute")
}
