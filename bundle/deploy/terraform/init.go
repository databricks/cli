package terraform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/libs/log"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type initialize struct{}

func (m *initialize) Name() string {
	return "terraform.Initialize"
}

func (m *initialize) findExecPath(ctx context.Context, b *bundle.Bundle, tf *config.Terraform) (string, error) {
	// If set, pass it through [exec.LookPath] to resolve its absolute path.
	if tf.ExecPath != "" {
		execPath, err := exec.LookPath(tf.ExecPath)
		if err != nil {
			return "", err
		}
		tf.ExecPath = execPath
		log.Debugf(ctx, "Using Terraform at %s", tf.ExecPath)
		return tf.ExecPath, nil
	}

	binDir, err := b.CacheDir("bin")
	if err != nil {
		return "", err
	}

	// If the execPath already exists, return it.
	execPath := filepath.Join(binDir, product.Terraform.BinaryName())
	_, err = os.Stat(execPath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err == nil {
		tf.ExecPath = execPath
		log.Debugf(ctx, "Using Terraform at %s", tf.ExecPath)
		return tf.ExecPath, nil
	}

	// Download Terraform to private bin directory.
	installer := &releases.LatestVersion{
		Product:     product.Terraform,
		Constraints: version.MustConstraints(version.NewConstraint("<2.0")),
		InstallDir:  binDir,
	}
	execPath, err = installer.Install(ctx)
	if err != nil {
		return "", fmt.Errorf("error downloading Terraform: %w", err)
	}

	tf.ExecPath = execPath
	log.Debugf(ctx, "Using Terraform at %s", tf.ExecPath)
	return tf.ExecPath, nil
}

func (m *initialize) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	tfConfig := b.Config.Bundle.Terraform
	if tfConfig == nil {
		tfConfig = &config.Terraform{}
		b.Config.Bundle.Terraform = tfConfig
	}

	execPath, err := m.findExecPath(ctx, b, tfConfig)
	if err != nil {
		return nil, err
	}

	workingDir, err := Dir(b)
	if err != nil {
		return nil, err
	}

	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		return nil, err
	}

	b.Terraform = tf
	return nil, nil
}

func Initialize() bundle.Mutator {
	return &initialize{}
}
