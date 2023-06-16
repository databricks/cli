package deployer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/locker"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type DeploymentStatus int

const (
	// Empty plan produced on terraform plan. No changes need to be applied
	NoChanges DeploymentStatus = iota

	// Deployment failed. No databricks assets were deployed
	Failed

	// Deployment failed/partially succeeded. failed to update remote terraform
	// state file.
	// The partially deployed resources are thus untracked and in most cases
	// will need to be cleaned up manually
	PartialButUntracked

	// Deployment failed/partially succeeded. Remote terraform state file is
	// updated with any partially deployed resources
	Partial

	// Deployment succeeded however the remote terraform state was not updated.
	// The deployed resources are thus untracked and in most cases will need to
	// be cleaned up manually
	CompleteButUntracked

	// Deployment succeeeded with remote terraform state file updated
	Complete
)

// Deployer is a struct to deploy a DAB to a databricks workspace
//
// Here's a high level description of what a deploy looks like:
//
// 1. Client compiles the bundle configuration to a terraform HCL config file
//
//  2. Client tries to acquire a lock on the remote root of the project.
//     -- If FAIL: print details about current holder of the deployment lock on
//     remote root and terminate deployment
//
// 3. Client reads terraform state from remote root
//
// 4. Client applies the diff in terraform config to the databricks workspace
//
// 5. Client updates terraform state file in remote root
//
// 6. Client releases the deploy lock on remote root
type Deployer struct {
	localRoot  string
	remoteRoot string
	env        string
	locker     *locker.Locker
	wsc        *databricks.WorkspaceClient
}

func Create(ctx context.Context, env, localRoot, remoteRoot string, wsc *databricks.WorkspaceClient) (*Deployer, error) {
	user, err := wsc.CurrentUser.Me(ctx)
	if err != nil {
		return nil, err
	}
	newLocker, err := locker.CreateLocker(user.UserName, remoteRoot, wsc)
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
	// Note: remote paths are scoped to `remoteRoot` through the locker. Also see [Create].
	return ".bundle/terraform.tfstate"
}

func (b *Deployer) tfStateLocalPath() string {
	return filepath.Join(b.DefaultTerraformRoot(), "terraform.tfstate")
}

func (d *Deployer) LoadTerraformState(ctx context.Context) error {
	r, err := d.locker.Read(ctx, d.tfStateRemotePath())
	if errors.Is(err, fs.ErrNotExist) {
		// If remote tf state is absent, use local tf state
		return nil
	}
	if err != nil {
		return err
	}
	err = os.MkdirAll(d.DefaultTerraformRoot(), os.ModeDir)
	if err != nil {
		return err
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return os.WriteFile(d.tfStateLocalPath(), b, os.ModePerm)
}

func (b *Deployer) SaveTerraformState(ctx context.Context) error {
	bytes, err := os.ReadFile(b.tfStateLocalPath())
	if err != nil {
		return err
	}
	return b.locker.Write(ctx, b.tfStateRemotePath(), bytes)
}

func (d *Deployer) Lock(ctx context.Context, isForced bool) error {
	return d.locker.Lock(ctx, isForced)
}

func (d *Deployer) Unlock(ctx context.Context) error {
	return d.locker.Unlock(ctx)
}

func (d *Deployer) ApplyTerraformConfig(ctx context.Context, configPath, terraformBinaryPath string, isForced bool) (DeploymentStatus, error) {
	applyErr := d.Lock(ctx, isForced)
	if applyErr != nil {
		return Failed, applyErr
	}
	defer func() {
		applyErr = d.Unlock(ctx)
		if applyErr != nil {
			log.Errorf(ctx, "failed to unlock deployment mutex: %s", applyErr)
		}
	}()

	applyErr = d.LoadTerraformState(ctx)
	if applyErr != nil {
		log.Debugf(ctx, "failed to load terraform state from workspace: %s", applyErr)
		return Failed, applyErr
	}

	tf, applyErr := tfexec.NewTerraform(configPath, terraformBinaryPath)
	if applyErr != nil {
		log.Debugf(ctx, "failed to construct terraform object: %s", applyErr)
		return Failed, applyErr
	}

	isPlanNotEmpty, applyErr := tf.Plan(ctx)
	if applyErr != nil {
		log.Debugf(ctx, "failed to compute terraform plan: %s", applyErr)
		return Failed, applyErr
	}

	if !isPlanNotEmpty {
		log.Debugf(ctx, "terraform plan returned a empty diff")
		return NoChanges, nil
	}

	applyErr = tf.Apply(ctx)
	// upload state even if apply fails to handle partial deployments
	saveStateErr := d.SaveTerraformState(ctx)

	if applyErr != nil && saveStateErr != nil {
		log.Errorf(ctx, "terraform apply failed: %s", applyErr)
		log.Errorf(ctx, "failed to upload terraform state after partial terraform apply: %s", saveStateErr)
		return PartialButUntracked, fmt.Errorf("deploymented failed: %s", applyErr)
	}
	if applyErr != nil {
		log.Errorf(ctx, "terraform apply failed: %s", applyErr)
		return Partial, fmt.Errorf("deploymented failed: %s", applyErr)
	}
	if saveStateErr != nil {
		log.Errorf(ctx, "failed to upload terraform state after completing terraform apply: %s", saveStateErr)
		return CompleteButUntracked, fmt.Errorf("failed to upload terraform state file: %s", saveStateErr)
	}
	return Complete, nil
}
