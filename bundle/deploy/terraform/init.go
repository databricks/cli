package terraform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/log"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
	"golang.org/x/exp/maps"
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

// This function sets temp dir location for terraform to use. If user does not
// specify anything here, we fall back to a `tmp` directory in the bundle's cache
// directory
//
// This is necessary to avoid trying to create temporary files in directories
// the CLI and its dependencies do not have access to.
//
// see: os.TempDir for more context
func setTempDirEnvVars(env map[string]string, b *bundle.Bundle) error {
	switch runtime.GOOS {
	case "windows":
		if v, ok := os.LookupEnv("TMP"); ok {
			env["TMP"] = v
		} else if v, ok := os.LookupEnv("TEMP"); ok {
			env["TEMP"] = v
		} else if v, ok := os.LookupEnv("USERPROFILE"); ok {
			env["USERPROFILE"] = v
		} else {
			tmpDir, err := b.CacheDir("tmp")
			if err != nil {
				return err
			}
			env["TMP"] = tmpDir
		}
	default:
		if v, ok := os.LookupEnv("TMPDIR"); ok {
			env["TMPDIR"] = v
		} else {
			// If TMPDIR is not set, we let the process fall back to its default value.
		}
	}
	return nil
}

func (m *initialize) Apply(ctx context.Context, b *bundle.Bundle) error {
	tfConfig := b.Config.Bundle.Terraform
	if tfConfig == nil {
		tfConfig = &config.Terraform{}
		b.Config.Bundle.Terraform = tfConfig
	}

	execPath, err := m.findExecPath(ctx, b, tfConfig)
	if err != nil {
		return err
	}

	workingDir, err := Dir(b)
	if err != nil {
		return err
	}

	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		return err
	}

	env, err := b.AuthEnv()
	if err != nil {
		return err
	}

	// Include $HOME in set of environment variables to pass along.
	home, ok := os.LookupEnv("HOME")
	if ok {
		env["HOME"] = home
	}

	// Set the temporary directory environment variables
	err = setTempDirEnvVars(env, b)
	if err != nil {
		return err
	}

	// Configure environment variables for auth for Terraform to use.
	log.Debugf(ctx, "Environment variables for Terraform: %s", strings.Join(maps.Keys(env), ", "))
	err = tf.SetEnv(env)
	if err != nil {
		return err
	}

	b.Terraform = tf
	return nil
}

func Initialize() bundle.Mutator {
	return &initialize{}
}
