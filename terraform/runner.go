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
	"context"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
)

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
