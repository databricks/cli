package terraform

import (
	"bytes"
	"context"
	"fmt"

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
	tf := b.Terraform
	if tf == nil {
		return fmt.Errorf("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}

	err = tf.Import(ctx, fmt.Sprintf("%s.%s", m.opts.ResourceType, m.opts.ResourceKey), m.opts.ResourceId)
	if err != nil {
		return fmt.Errorf("terraform import: %w", err)
	}

	buf := bytes.NewBuffer(nil)
	tf.SetStdout(buf)
	changed, err := tf.Plan(ctx)
	if err != nil {
		return fmt.Errorf("terraform plan: %w", err)
	}

	if changed && !m.opts.AutoApprove {
		cmdio.LogString(ctx, buf.String())
		ans, err := cmdio.AskYesOrNo(ctx, "Confirm import changes? Changes will be remotely only after running 'bundle deploy'.")
		if err != nil {
			return err
		}
		if !ans {
			err = tf.StateRm(ctx, fmt.Sprintf("%s.%s", m.opts.ResourceType, m.opts.ResourceKey))
			if err != nil {
				return err
			}
			return fmt.Errorf("import aborted")
		}
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

type unbind struct {
	resourceType string
	resourceKey  string
}

func (m *unbind) Apply(ctx context.Context, b *bundle.Bundle) error {
	tf := b.Terraform
	if tf == nil {
		return fmt.Errorf("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}

	err = tf.StateRm(ctx, fmt.Sprintf("%s.%s", m.resourceType, m.resourceKey))
	if err != nil {
		return fmt.Errorf("terraform state rm: %w", err)
	}

	return nil
}

func (*unbind) Name() string {
	return "terraform.Unbind"
}

func Unbind(resourceType string, resourceKey string) bundle.Mutator {
	return &unbind{resourceType: resourceType, resourceKey: resourceKey}
}
