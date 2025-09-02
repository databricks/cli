package terraform

import (
	"context"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
)

// Installer is an interface that can be used to install a Terraform binary.
// It exists to facilitate testing.
type Installer interface {
	Install(ctx context.Context, dir string, version *version.Version) (string, error)
}

// tfInstaller is a real installer that uses the HashiCorp installer library.
type tfInstaller struct{}

// Install installs a Terraform binary using the HashiCorp installer library.
func (i tfInstaller) Install(ctx context.Context, dir string, version *version.Version) (string, error) {
	installer := &releases.ExactVersion{
		Product:    product.Terraform,
		Version:    version,
		InstallDir: dir,
		Timeout:    1 * time.Minute,
	}
	return installer.Install(ctx)
}
