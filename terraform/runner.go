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
	"log"
	"os"

	"github.com/databricks/bricks/project"
	"github.com/databricks/bricks/utilities"
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

// returns location of terraform state on DBFS based on project's deployment isolation level.
func (d *TerraformDeployer) remoteTfstateLoc() string {
	prefix := project.Current.DeploymentIsolationPrefix()
	return fmt.Sprintf("%s/%s/terraform.tfstate", DeploymentStateRemoteLocation, prefix)
}

// returns structured representation of terraform state on DBFS.
func (d *TerraformDeployer) remoteState(ctx context.Context) (*tfjson.State, int, error) {
	raw, err := utilities.ReadDbfsFile(ctx,
		project.Current.WorkspacesClient(),
		d.remoteTfstateLoc(),
	)
	if err != nil {
		return nil, 0, err
	}
	return d.tfstateFromReader(bytes.NewBuffer(raw))
}

// opens file handle for local-backend terraform state, that has to be closed in the calling
// methods. this file alone is not the authoritative state of deployment and has to properly
// be synced with remote counterpart.
func (d *TerraformDeployer) openLocalState() (*os.File, error) {
	return os.Open(fmt.Sprintf("%s/terraform.tfstate", d.WorkDir))
}

// returns structured representation of terraform state on local machine. as part of
// the optimistic concurrency control, please make sure to always compare the serial
// number of local and remote states before proceeding with deployment.
func (d *TerraformDeployer) localState() (*tfjson.State, int, error) {
	local, err := d.openLocalState()
	if err != nil {
		return nil, 0, err
	}
	defer local.Close()
	return d.tfstateFromReader(local)
}

// converts input stream into structured representation of terraform state and deployment
// serial number, that helps controlling versioning and synchronisation via optimistic locking.
func (d *TerraformDeployer) tfstateFromReader(reader io.Reader) (*tfjson.State, int, error) {
	var state tfjson.State
	state.UseJSONNumber(true)
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	err := decoder.Decode(&state)
	if err != nil {
		return nil, 0, err
	}
	err = state.Validate()
	if err != nil {
		return nil, 0, err
	}
	var serialWrapper struct {
		Serial int `json:"serial,omitempty"`
	}
	// TODO: use byte buffer if this decoder fails on double reading
	err = decoder.Decode(&serialWrapper)
	if err != nil {
		return nil, 0, err
	}
	return &state, serialWrapper.Serial, nil
}

// uploads terraform state from local directory to designated DBFS location.
func (d *TerraformDeployer) uploadTfstate(ctx context.Context) error {
	local, err := d.openLocalState()
	if err != nil {
		return err
	}
	defer local.Close()
	raw, err := io.ReadAll(local)
	if err != nil {
		return err
	}
	// TODO: make sure that deployment locks are implemented
	return utilities.CreateDbfsFile(ctx,
		project.Current.WorkspacesClient(),
		d.remoteTfstateLoc(),
		raw,
		true,
	)
}

// downloads terraform state from DBFS to local working directory.
func (d *TerraformDeployer) downloadTfstate(ctx context.Context) error {
	remote, serialDeployed, err := d.remoteState(ctx)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] remote serial is %d", serialDeployed)
	local, err := d.openLocalState()
	if err != nil {
		return err
	}
	defer local.Close()
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
