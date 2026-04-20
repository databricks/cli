package terraform

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
)

// Installer resolves a Terraform binary on disk. Split out behind an
// interface so tests can stub the download step.
type Installer interface {
	// Install downloads terraform v into dir and returns the absolute path
	// of the installed binary.
	Install(ctx context.Context, dir string, v *version.Version) (string, error)
}

// hcInstaller is the production Installer — delegates to hashicorp/hc-install.
type hcInstaller struct{}

// Install downloads terraform via hashicorp/hc-install.
func (hcInstaller) Install(ctx context.Context, dir string, v *version.Version) (string, error) {
	installer := &releases.ExactVersion{
		Product:    product.Terraform,
		Version:    v,
		InstallDir: dir,
		Timeout:    1 * time.Minute,
	}
	return installer.Install(ctx)
}

// lookupVersionFromEnv returns the terraform version configured via
// DATABRICKS_TF_VERSION, or (nil, false, nil) if unset.
func lookupVersionFromEnv(ctx context.Context) (*version.Version, bool, error) {
	raw, ok := env.Lookup(ctx, VersionEnv)
	if !ok || raw == "" {
		return nil, false, nil
	}
	v, err := version.NewVersion(raw)
	if err != nil {
		return nil, false, fmt.Errorf("parse %s=%q: %w", VersionEnv, raw, err)
	}
	return v, true, nil
}
