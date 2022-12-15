package terraform

import (
	"context"
	"os/exec"

	"github.com/databricks/bricks/bundle"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type initialize struct{}

func (m *initialize) Name() string {
	return "terraform.Initialize"
}

func (m *initialize) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	workingDir, err := Dir(b)
	if err != nil {
		return nil, err
	}

	execPath, err := exec.LookPath("terraform")
	if err != nil {
		return nil, err
	}

	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		return nil, err
	}

	b.Terraform = tf
	return nil, nil
}

func Initialize() bundle.Mutator {
	return &initialize{}
}
