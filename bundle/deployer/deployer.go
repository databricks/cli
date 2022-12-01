package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type DeploymentStatus int

const (
	// Empty plan produced on terraform plan. No changes need to be applied
	NoChanges DeploymentStatus = iota
	// Deployment failed. No databricks assets were deployed
	Failed
	// Deployment failed/partially suceeeded. failed to update remote terraform
	// state file.
	// The partially deployed resources are thus untracked and in most cases
	// will need to be cleaned up manually
	PartialButUntracked
	// Deployment failed/partially suceeeded. Remote terraform state file is
	// updated with any partially deployed resources
	Partial
	// Deployment suceeeded however the remote terraform state was not updated.
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
// 2. Client tries to acquire a lock on the remote root of the project.
//  	-- If FAIL: print details about current holder of the deployment lock on
//					remote root and terminate deployment
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
	return path.Join(b.remoteRoot, ".bundle", "terraform.tfstate")
}

func (b *Deployer) tfStateLocalPath() string {
	return filepath.Join(b.DefaultTerraformRoot(), "terraform.tfstate")
}

func (b *Deployer) LoadTerraformState(ctx context.Context) error {
	res, err := b.locker.GetJsonFileContent(ctx, b.wsc, b.tfStateRemotePath())
	if err != nil {
		// If remote tf state is absent, use local tf state
		if strings.Contains(err.Error(), "File not found.") {
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
	err = os.WriteFile(b.tfStateLocalPath(), bytes, os.ModePerm)
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
		log.Printf("[DEBUG] failed to load terraform state from workspace: %s", err)
		return Failed, err
	}

	tf, err := tfexec.NewTerraform(configPath, terraformBinaryPath)
	if err != nil {
		log.Printf("[DEBUG] failed to construct terraform object: %s", err)
		return Failed, err
	}

	isPlanNotEmpty, err := tf.Plan(ctx)
	if err != nil {
		log.Printf("[DEBUG] failed to compute terraform plan: %s", err)
		return Failed, err
	}

	if !isPlanNotEmpty {
		log.Printf("[DEBUG] terraform plan returned a empty diff")
		return NoChanges, nil
	}

	err = tf.Apply(ctx)
	// upload state even if apply fails to handle partial deployments
	err2 := d.SaveTerraformState(ctx)

	if err != nil && err2 != nil {
		log.Printf("[ERROR] terraform apply failed: %s", err)
		log.Printf("[ERROR] failed to upload terraform state after partial terraform apply: %s", err2)
		return PartialButUntracked, fmt.Errorf("deploymented failed: %s", err)
	}
	if err != nil {
		log.Printf("[ERROR] terraform apply failed: %s", err)
		return Partial, fmt.Errorf("deploymented failed: %s", err)
	}
	if err2 != nil {
		log.Printf("[ERROR] failed to upload terraform state after completing terraform apply: %s", err2)
		return CompleteButUntracked, fmt.Errorf("failed to upload terraform state file: %s", err2)
	}
	return Complete, nil
}
