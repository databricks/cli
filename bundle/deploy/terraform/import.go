package terraform

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
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
func (m *importResource) Apply(ctx context.Context, b *bundle.Bundle) error {
	dir, err := Dir(ctx, b)
	if err != nil {
		return err
	}

	tf := b.Terraform
	if tf == nil {
		return fmt.Errorf("terraform not initialized")
	}

	err = tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}
	tmpDir, err := os.MkdirTemp("", "state-*")
	if err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}
	tmpState := filepath.Join(tmpDir, TerraformStateFileName)

	importAddress := fmt.Sprintf("%s.%s", m.opts.ResourceType, m.opts.ResourceKey)
	err = tf.Import(ctx, importAddress, m.opts.ResourceId, tfexec.StateOut(tmpState))
	if err != nil {
		return fmt.Errorf("terraform import: %w", err)
	}

	buf := bytes.NewBuffer(nil)
	tf.SetStdout(buf)

	//lint:ignore SA1019 We use legacy -state flag for now to plan the import changes based on temporary state file
	changed, err := tf.Plan(ctx, tfexec.State(tmpState), tfexec.Target(importAddress))
	if err != nil {
		return fmt.Errorf("terraform plan: %w", err)
	}

	defer os.RemoveAll(tmpDir)

	if changed && !m.opts.AutoApprove {
		output := buf.String()
		// Remove output starting from Warning until end of output
		output = output[:bytes.Index([]byte(output), []byte("Warning:"))]
		cmdio.LogString(ctx, output)
		ans, err := cmdio.AskYesOrNo(ctx, "Confirm import changes? Changes will be remotely applied only after running 'bundle deploy'.")
		if err != nil {
			return err
		}
		if !ans {
			return fmt.Errorf("import aborted")
		}
	}

	// If user confirmed changes, move the state file from temp dir to state location
	f, err := os.Create(filepath.Join(dir, TerraformStateFileName))
	if err != nil {
		return err
	}
	defer f.Close()

	tmpF, err := os.Open(tmpState)
	if err != nil {
		return err
	}
	defer tmpF.Close()

	_, err = io.Copy(f, tmpF)
	if err != nil {
		return err
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
