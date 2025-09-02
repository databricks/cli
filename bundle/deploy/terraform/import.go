package terraform

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type BindOptions struct {
	AutoApprove  bool
	ResourceType string
	ResourceKey  string
	ResourceId   string
}

type importResource struct {
	opts *BindOptions
}

// Apply implements bundle.Mutator.
func (m *importResource) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.DirectDeployment {
		return diag.Errorf("import is not implemented for DATABRICKS_CLI_DEPLOYMENT=direct")
	}

	dir, err := Dir(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	tf := b.Terraform
	if tf == nil {
		return diag.Errorf("terraform not initialized")
	}

	err = tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return diag.Errorf("terraform init: %v", err)
	}
	tmpDir, err := os.MkdirTemp("", "state-*")
	if err != nil {
		return diag.Errorf("terraform init: %v", err)
	}
	tmpState := filepath.Join(tmpDir, b.StateFilename())

	importAddress := fmt.Sprintf("%s.%s", m.opts.ResourceType, m.opts.ResourceKey)
	err = tf.Import(ctx, importAddress, m.opts.ResourceId, tfexec.StateOut(tmpState))
	if err != nil {
		return diag.Errorf("terraform import: %v", err)
	}

	buf := bytes.NewBuffer(nil)
	tf.SetStdout(buf)

	//nolint:staticcheck // SA1019 We use legacy -state flag for now to plan the import changes based on temporary state file
	changed, err := tf.Plan(ctx, tfexec.State(tmpState), tfexec.Target(importAddress))
	if err != nil {
		return diag.Errorf("terraform plan: %v", err)
	}

	defer os.RemoveAll(tmpDir)

	if changed && !m.opts.AutoApprove {
		output := buf.String()
		// Remove output starting from Warning until end of output
		output = output[:strings.Index(output, "Warning:")]
		cmdio.LogString(ctx, output)

		if !cmdio.IsPromptSupported(ctx) {
			return diag.Errorf("This bind operation requires user confirmation, but the current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed.")
		}

		ans, err := cmdio.AskYesOrNo(ctx, "Confirm import changes? Changes will be remotely applied only after running 'bundle deploy'.")
		if err != nil {
			return diag.FromErr(err)
		}
		if !ans {
			return diag.Errorf("import aborted")
		}
	}

	// If user confirmed changes, move the state file from temp dir to state location
	f, err := os.Create(filepath.Join(dir, b.StateFilename()))
	if err != nil {
		return diag.FromErr(err)
	}
	defer f.Close()

	tmpF, err := os.Open(tmpState)
	if err != nil {
		return diag.FromErr(err)
	}
	defer tmpF.Close()

	_, err = io.Copy(f, tmpF)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// Name implements bundle.Mutator.
func (*importResource) Name() string {
	return "terraform.Import"
}

func Import(opts *BindOptions) bundle.Mutator {
	return &importResource{opts: opts}
}
