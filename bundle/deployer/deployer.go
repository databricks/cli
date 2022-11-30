package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/bricks/utilities"
	"github.com/databricks/databricks-sdk-go"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type DeploymentStatus int

const (
	NoChanges DeploymentStatus = iota
	Failed
	Partial
	Success
)

type Deployer struct {
	localRoot  string
	remoteRoot string
	env        string
	locker     *Locker
	wsc        *databricks.WorkspaceClient
}

func Create(ctx context.Context, env, localRoot, remoteRoot string, wsc *databricks.WorkspaceClient) (*Deployer, error) {
	user, err := wsc.CurrentUser.Me(ctx)
	if err != nil {
		return nil, err
	}
	newLocker := CreateLocker(user.UserName, remoteRoot)
	if err != nil {
		return nil, err
	}
	return &Deployer{
		localRoot:  localRoot,
		remoteRoot: remoteRoot,
		env:        env,
		locker:     newLocker,
		wsc:        wsc,
	}, nil
}

func (b *Deployer) DefaultTerraformRoot() string {
	return filepath.Join(b.localRoot, ".databricks/bundle", b.env)
}

func (b *Deployer) tfStateRemotePath() string {
	return filepath.Join(b.remoteRoot, ".bundle", "terraform.tfstate")
}

func (b *Deployer) tfStateLocalPath() string {
	return filepath.Join(b.DefaultTerraformRoot(), "terraform.tfstate")
}

func (b *Deployer) LoadTerraformState(ctx context.Context) error {
	res, err := utilities.GetJsonFileContent(ctx, b.wsc, b.tfStateRemotePath())
	if err != nil {
		// remote tf state is the source of truth. If it's absent, we delete the
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
	err = os.MkdirAll(b.DefaultTerraformRoot(), os.ModeDir)
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

func (b *Deployer) SaveTerraformState(ctx context.Context) error {
	bytes, err := os.ReadFile(b.tfStateLocalPath())
	if err != nil {
		return err
	}
	return b.locker.PutFile(ctx, b.wsc, b.tfStateRemotePath(), bytes)
}

func (b *Deployer) Lock(ctx context.Context) error {
	return b.locker.Lock(ctx, b.wsc, false)
}

func (b *Deployer) Unlock(ctx context.Context) error {
	return b.locker.Unlock(ctx, b.wsc)
}

func (d *Deployer) ApplyTerraformConfig(ctx context.Context, configPath, terraformBinaryPath string) (DeploymentStatus, error) {
	err := d.locker.Lock(ctx, d.wsc, false)
	if err != nil {
		return Failed, err
	}
	defer func() {
		err = d.locker.Unlock(ctx, d.wsc)
		if err != nil {
			log.Printf("[ERROR] failed to unlock deployment mutex: %s", err)
		}
	}()

	err = d.LoadTerraformState(ctx)
	if err != nil {
		return Failed, err
	}

	tf, err := tfexec.NewTerraform(configPath, terraformBinaryPath)
	if err != nil {
		return Failed, err
	}

	isPlanNotEmpty, err := tf.Plan(ctx)
	if err != nil {
		return Failed, err
	}

	if !isPlanNotEmpty {
		// log.Printf("[INFO] state diff is empty. No changes applied")
		return NoChanges, nil
	}

	err = tf.Apply(ctx)
	// upload state even if apply fails to handle partial deployments
	err2 := d.SaveTerraformState(ctx)
	if err != nil {
		return Partial, fmt.Errorf("deploymented failed: %s", err)
	}
	if err2 != nil {
		return Partial, fmt.Errorf("failed to upload updated tfstate file: %s", err2)
	}
	return Success, nil

}
