/*
How to simplify terraform configuration for the project?
---

Solve the following adoption slowers:

- remove the need for `required_providers` block
- authenticate Databricks provider with the same DatabricksClient
- skip downloading and locking Databricks provider every time (few seconds)
- users won't have to copy-paste these into their configs:

```hcl
terraform {
  required_providers {
    databricks = {
      source  = "databrickslabs/databricks"
    }
  }
}

provider "databricks" {
}
```

Terraform Plugin SDK v2 is using similar techniques for testing providers. One may find
details in github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource/plugin.go. In short:

- init provider isntance
- start terraform plugin GRPC server
- "reattach" providers and specify the `tfexec.Reattach` options, which essentially
  forward GRPC address to terraform subprocess.
- this can be done by either adding a source depenency on Databricks provider
  or adding a special launch mode to it.

For now
---
Let's see how far we can get without GRPC magic.
*/
package terraform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/databricks/bricks/project"
	"github.com/databrickslabs/terraform-provider-databricks/storage"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"

	tfjson "github.com/hashicorp/terraform-json"
)

const DeploymentStateRemoteLocation = "dbfs:/FileStore/deployment-state"

type TerraformDeployer struct {
	WorkDir string
	CopyTfs bool

	tf *tfexec.Terraform
}

func (d *TerraformDeployer) Init(ctx context.Context) error {
	if d.CopyTfs {
		panic("copying tf configuration files to a temporary dir not yet implemented")
	}
	// TODO: most likely merge the methods
	exec, err := newTerraform(ctx, d.WorkDir, map[string]string{})
	if err != nil {
		return err
	}
	d.tf = exec
	return nil
}

func (d *TerraformDeployer) remoteTfstateLoc() string {
	prefix := project.Current.DeploymentIsolationPrefix()
	return fmt.Sprintf("%s/%s/terraform.tfstate", DeploymentStateRemoteLocation, prefix)
}

func (d *TerraformDeployer) remoteState(ctx context.Context) (*tfjson.State, error) {
	dbfs := storage.NewDbfsAPI(ctx, project.Current.Client())
	raw, err := dbfs.Read(d.remoteTfstateLoc())
	if err != nil {
		return nil, err
	}
	return d.tfstateFromReader(bytes.NewBuffer(raw))
}

func (d *TerraformDeployer) openLocalState() (*os.File, error) {
	return os.Open(fmt.Sprintf("%s/terraform.tfstate", d.WorkDir))
}

func (d *TerraformDeployer) localState() (*tfjson.State, error) {
	raw, err := d.openLocalState()
	if err != nil {
		return nil, err
	}
	return d.tfstateFromReader(raw)
}

func (d *TerraformDeployer) tfstateFromReader(reader io.Reader) (*tfjson.State, error) {
	var state tfjson.State
	state.UseJSONNumber(true)
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	err := decoder.Decode(&state)
	if err != nil {
		return nil, err
	}
	err = state.Validate()
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (d *TerraformDeployer) uploadTfstate(ctx context.Context) error {
	// scripts/azcli-integration/terraform.tfstate
	dbfs := storage.NewDbfsAPI(ctx, project.Current.Client())
	f, err := d.openLocalState()
	if err != nil {
		return err
	}
	raw, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	// TODO: make sure that deployment locks are implemented
	return dbfs.Create(d.remoteTfstateLoc(), raw, true)
}

func (d *TerraformDeployer) downloadTfstate(ctx context.Context) error {
	remote, err := d.remoteState(ctx)
	if err != nil {
		return err
	}
	local, err := d.openLocalState()
	if err != nil {
		return err
	}
	raw, err := json.Marshal(remote)
	if err != nil {
		return err
	}
	_, err = io.Copy(local, bytes.NewBuffer(raw))
	return err
}

// installs terraform to a temporary directory (for now)
func installTerraform(ctx context.Context) (string, error) {
	// TODO: let configuration and/or environment variable specify
	// terraform binary. Or detect if terraform is installed in the $PATH
	installer := &releases.ExactVersion{
		Product: product.Terraform,
		Version: version.Must(version.NewVersion("1.1.0")),
	}
	return installer.Install(ctx)
}

func newTerraform(ctx context.Context, workDir string, env map[string]string) (*tfexec.Terraform, error) {
	execPath, err := installTerraform(ctx)
	if err != nil {
		return nil, err
	}
	// TODO: figure out how to cleanup/skip `.terraform*` files and dirs, not to confuse users
	// one of the options: take entire working directory with *.tf files and move them to tmpdir.
	// make it optional, of course, otherwise debugging may become super hard.
	tf, err := tfexec.NewTerraform(workDir, execPath)
	if err != nil {
		return nil, err
	}
	err = tf.SetEnv(env)
	if err != nil {
		return nil, err
	}
	return tf, err
}
