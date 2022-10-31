package script

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/bricks/lib/flavor"
	"github.com/databricks/bricks/lib/spawn"
	"github.com/databricks/databricks-sdk-go/databricks"
)

type Script struct {
	OnInit   string `json:"init,omitempty"`
	OnDeploy string `json:"deploy,omitempty"`
}

// Detected returns true if any if the scripts are configured
func (s *Script) Detected(p flavor.Project) bool {
	return s.OnInit != "" || s.OnDeploy != ""
}

// TODO: move to a separate interface
func (s *Script) LocalArtifacts(ctx context.Context, p flavor.Project) (flavor.Artifacts, error) {
	return nil, nil
}

func (s *Script) Build(ctx context.Context, p flavor.Project, status func(string)) error {
	if s.OnDeploy == "" {
		return nil
	}
	cfg := p.WorkspacesClient().Config
	err := cfg.EnsureResolved()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	for _, a := range databricks.ConfigAttributes {
		if len(a.EnvVars) != 1 {
			continue
		}
		v := a.GetString(cfg)
		if v == "" {
			continue
		}
		// set environment variables of the current process to propagate
		// the authentication credentials
		os.Setenv(a.EnvVars[0], v)
	}
	out, err := spawn.ExecAndPassErr(ctx, "/bin/sh", "-c", s.OnDeploy)
	if err != nil {
		println(string(out))
		return fmt.Errorf("failed: %s", s.OnDeploy)
	}
	return nil
}
