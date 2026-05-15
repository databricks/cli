// Package pipelineenv calls pipelines.ApplyEnvironment after bundle deploy
// to refresh dev-mode pipelines when pyproject.toml or a wheel artifact changes.
package pipelineenv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

	newHash, err := envInputHash(b)
	if err != nil {
		log.Debugf(ctx, "pipelineenv: skipping; could not hash env inputs: %v", err)
		return nil
	}
	prevHash := readPrevHash(ctx, b)
	if newHash == prevHash {
		return nil
	}

	cmdio.LogString(ctx, "Change detected to Python environment: running 'databricks pipelines apply-environment' to update pipeline(s)...")

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
	if allOK {
		if err := writePrevHash(ctx, b, newHash); err != nil {
			log.Debugf(ctx, "pipelineenv: failed to persist hash: %v", err)
		}
	}
	return nil
}

// pipelinesNeedingEnvApply returns the dev-mode pipelines that could be
// affected by the SDP env cache. Skipped:
//   - missing/not-yet-deployed (no ID)
//   - production mode (env always reinstalls on every update)
func pipelinesNeedingEnvApply(b *bundle.Bundle) []*resources.Pipeline {
	keys := make([]string, 0, len(b.Config.Resources.Pipelines))
	for k := range b.Config.Resources.Pipelines {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	out := make([]*resources.Pipeline, 0, len(keys))
	for _, k := range keys {
		p := b.Config.Resources.Pipelines[k]
		if p == nil || p.ID == "" || !p.Development {
			continue
		}
		out = append(out, p)
	}
	return out
}

// envInputHash returns a sha256 of every file that affects the pipeline's
// installed Python environment: pyproject.toml at the bundle root plus the
// pre-patchwheel content of each whl artifact. Patched wheels are skipped
// because patchwheel embeds an mtime suffix that changes on every build.
func envInputHash(b *bundle.Bundle) (string, error) {
	h := sha256.New()
	if err := hashFileInto(h, filepath.Join(b.BundleRootPath, "pyproject.toml"), "pyproject.toml"); err != nil {
		return "", err
	}
	keys := make([]string, 0, len(b.Config.Artifacts))
	for k := range b.Config.Artifacts {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		art := b.Config.Artifacts[k]
		if art == nil || art.Type != config.ArtifactPythonWheel {
			continue
		}
		for _, f := range art.Files {
			if err := hashFileInto(h, f.Source, "wheel:"+filepath.Base(f.Source)); err != nil {
				return "", err
			}
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashFileInto(h io.Writer, path, label string) error {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer f.Close()
	if _, err := io.WriteString(h, label+"\n"); err != nil {
		return err
	}
	_, err = io.Copy(h, f)
	return err
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
