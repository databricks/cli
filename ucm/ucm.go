// Package ucm is the engine layer for the `databricks ucm` subcommand. It
// parallels the `bundle` package and reuses shared primitives from libs/*.
//
// M0 includes just enough of the engine to support `validate` and `schema`:
// a typed/dynamic config tree, a mutator plumbing layer, a filesystem loader,
// and a tag-validation mutator. Terraform/state/deploy wiring lands in M1.
package ucm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/folders"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/databricks-sdk-go"
)

// RootEnv is the environment variable that pins a ucm root directory,
// overriding filesystem traversal. Mirrors DATABRICKS_BUNDLE_ROOT.
const RootEnv = "DATABRICKS_UCM_ROOT"

// Ucm is the in-memory representation of a ucm deployment.
type Ucm struct {
	// RootPath is the local path to the directory containing ucm.yml.
	RootPath string

	// Config is the loaded configuration tree.
	Config config.Root

	// Target is the snapshot of the selected target after SelectTarget runs.
	// nil until a target has been selected.
	Target *config.Target

	// CurrentUser is populated by the PopulateCurrentUser mutator from the
	// workspace client's CurrentUser.Me(). Lives on Ucm (not Config) so the
	// dyn-tree round-trip in MarkMutatorEntry/Exit doesn't zero it between
	// mutator applies.
	CurrentUser *config.User

	// getClient memoizes the workspace client built from Config.Workspace.
	// Initialized lazily by WorkspaceClientE via initClientOnce.
	getClient func() (*databricks.WorkspaceClient, error)

	Metrics Metrics
}

// Load builds a Ucm for the given root path, reading the ucm.yml file
// underneath it.
func Load(ctx context.Context, path string) (*Ucm, error) {
	configFile, err := config.FileNames.FindInPath(path)
	if err != nil {
		return nil, err
	}

	cfg, diags := config.Load(configFile)
	if diags.HasError() {
		return nil, diags.Error()
	}

	log.Debugf(ctx, "Found ucm root at %s (file %s)", path, configFile)

	u := &Ucm{
		RootPath: filepath.Clean(path),
		Config:   *cfg,
	}
	return u, nil
}

// MustLoad finds the ucm root, loads the config, and logs any error through
// logdiag. Callers should check logdiag.HasError(ctx) afterwards.
func MustLoad(ctx context.Context) *Ucm {
	root, err := mustGetRoot(ctx)
	if err != nil {
		logdiag.LogError(ctx, err)
		return nil
	}
	u, err := Load(ctx, root)
	if err != nil {
		logdiag.LogError(ctx, err)
		return nil
	}
	return u
}

func getRootEnv() (string, error) {
	path := os.Getenv(RootEnv)
	if path == "" {
		return "", nil
	}
	stat, err := os.Stat(path)
	if err == nil && !stat.IsDir() {
		err = errors.New("not a directory")
	}
	if err != nil {
		return "", fmt.Errorf(`invalid ucm root %s=%q: %w`, RootEnv, path, err)
	}
	return path, nil
}

func getRootWithTraversal() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for _, name := range config.FileNames {
		if path, err := folders.FindDirWithLeaf(wd, name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("unable to locate ucm root: %s not found", config.FileNames[0])
}

func mustGetRoot(_ context.Context) (string, error) {
	if path, err := getRootEnv(); path != "" || err != nil {
		return path, err
	}
	return getRootWithTraversal()
}
