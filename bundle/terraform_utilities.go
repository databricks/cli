package bundle

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/bricks/lock"
	"github.com/databricks/bricks/utilities"
	"github.com/hashicorp/terraform-exec/tfexec"
)

func CreateBundle(env, localRoot, remoteRoot, terraformBinaryPath string) *Bundle {
	return &Bundle{
		localRoot:           localRoot,
		remoteRoot:          remoteRoot,
		env:                 env,
		terraformBinaryPath: terraformBinaryPath,
	}
}

func (b *Bundle) Locker() (*lock.DeployLocker, error) {
	if b.locker != nil {
		return b.locker, nil
	}
	user, err := b.User()
	if err != nil {
		return nil, err
	}
	newLocker, err := lock.CreateLocker(user, false, b.remoteRoot)
	if err != nil {
		return nil, err
	}
	b.locker = newLocker
	return b.locker, nil
}

func (b *Bundle) User() (string, error) {
	if b.user != "" {
		return b.user, nil
	}
	user, err := b.WorkspaceClient().CurrentUser.Me(context.Background())
	if err != nil {
		return "", err
	}
	b.user = user.UserName
	return b.user, nil
}

func (b *Bundle) cacheDir() string {
	return filepath.Join(b.localRoot, ".databricks/bundle")
}

func (b *Bundle) tfHclPath() string {
	return filepath.Join(b.cacheDir(), b.env, "main.tf")
}

func (b *Bundle) tfStateRemotePath() string {
	return filepath.Join(b.remoteRoot, ".bundle", "terraform.tfstate")
}

func (b *Bundle) tfStateLocalPath() string {
	return filepath.Join(b.cacheDir(), b.env, "terraform.tfstate")
}

func (b *Bundle) ExportTerraformState(ctx context.Context) error {
	res, err := utilities.GetFileContent(ctx, b.WorkspaceClient(), b.tfStateRemotePath())
	if err != nil {
		// remote tf state is the source of truth. If its absent, we delete the
		// local state too and start from a clean slate
		if strings.Contains(err.Error(), "File not found.") {
			err := os.Remove(b.tfStateLocalPath())
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete local tf state: %s", err)
			}
			return nil
		} else {
			return err
		}
	}
	bytes, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	err = os.MkdirAll(b.cacheDir(), os.ModeDir)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(b.tfStateLocalPath(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	_, err = f.Write(bytes)
	return err
}

func (b *Bundle) ImportTerraformState(ctx context.Context) error {
	l, err := b.Locker()
	if err != nil {
		return err
	}
	bytes, err := os.ReadFile(b.tfStateLocalPath())
	if err != nil {
		return err
	}
	return l.SafePutFile(ctx, b.WorkspaceClient(), b.tfStateRemotePath(), bytes)
}

func (b *Bundle) Lock(ctx context.Context) error {
	l, err := b.Locker()
	if err != nil {
		return err
	}
	return l.Lock(ctx, b.WorkspaceClient())
}

func (b *Bundle) Unlock(ctx context.Context) error {
	l, err := b.Locker()
	if err != nil {
		return err
	}
	return l.Unlock(ctx, b.WorkspaceClient())
}

func (b *Bundle) GetTerraformHandle(ctx context.Context) (*tfexec.Terraform, error) {
	return tfexec.NewTerraform(filepath.Dir(b.tfHclPath()), b.terraformBinaryPath)
}
