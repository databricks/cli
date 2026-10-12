package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// SetupTerraform installs Terraform and the Databricks Terraform provider into a
// local filesystem mirror by invoking acceptance/install_terraform.py, then exports
// TF_CLI_CONFIG_FILE / DATABRICKS_TF_CLI_CONFIG_FILE / DATABRICKS_TF_EXEC_PATH /
// TERRAFORM so the CLI subprocess used by tests resolves the provider locally
// instead of contacting registry.terraform.io.
//
// Intended for use from TestMain in integration test packages: the setup runs
// once per `go test` invocation and behaves the same in CI (where
// registry.terraform.io is blocked by Databricks corp network policy) and on a
// developer's laptop.
func SetupTerraform() error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("locate repo root: %w", err)
	}

	scriptPath := filepath.Join(repoRoot, "acceptance", "install_terraform.py")
	buildDir := filepath.Join(repoRoot, "acceptance", "build", runtime.GOOS+"_"+runtime.GOARCH)
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return fmt.Errorf("create build dir %s: %w", buildDir, err)
	}

	cmd := exec.Command("python3", scriptPath, "--targetdir", buildDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", scriptPath, err)
	}

	binarySuffix := ""
	if runtime.GOOS == "windows" {
		binarySuffix = ".exe"
	}
	terraformrcPath := filepath.Join(buildDir, ".terraformrc")
	terraformExecPath := filepath.Join(buildDir, "terraform"+binarySuffix)

	envs := map[string]string{
		"TF_CLI_CONFIG_FILE":            terraformrcPath,
		"DATABRICKS_TF_CLI_CONFIG_FILE": terraformrcPath,
		"DATABRICKS_TF_EXEC_PATH":       terraformExecPath,
		"TERRAFORM":                     terraformExecPath,
	}
	for k, v := range envs {
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("set %s: %w", k, err)
		}
	}
	return nil
}

// findRepoRoot walks up from the current working directory until it finds a
// directory that contains both go.mod and acceptance/install_terraform.py. The
// pair uniquely identifies the cli repo root and avoids stopping at the nested
// tools/go.mod or any other module along the way.
func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		_, modErr := os.Stat(filepath.Join(dir, "go.mod"))
		_, scriptErr := os.Stat(filepath.Join(dir, "acceptance", "install_terraform.py"))
		if modErr == nil && scriptErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("cli repo root not found searching up from %s", cwd)
		}
		dir = parent
	}
}
