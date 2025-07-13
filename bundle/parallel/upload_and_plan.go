package parallel

import (
	"context"
	"errors"
	"io/fs"
	"sync"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/logdiag"
	synclib "github.com/databricks/cli/libs/sync"
)

type uploadAndPlan struct {
	syncOutputHandler synclib.OutputHandler
}

func UploadAndPlan(syncOutputHandler synclib.OutputHandler) bundle.Mutator {
	return &uploadAndPlan{syncOutputHandler}
}

func (m *uploadAndPlan) Name() string {
	return "parallel.uploadAndPlan"
}

func (m *uploadAndPlan) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if !b.DirectDeployment {
		bundle.ApplySeqContext(ctx, b,
			terraform.Interpolate(),
			terraform.Write(),
		)
	}

	syncOpts, err := files.GetSyncOptions(ctx, b)
	syncOpts.OutputHandler = m.syncOutputHandler
	if err != nil {
		logdiag.LogError(ctx, err)
		return nil
	}

	tfDir, err := terraform.Dir(ctx, b)
	if err != nil {
		logdiag.LogError(ctx, err)
		return nil
	}
	// Compute terraform plan and upload files at the same time.
	// These operations are independent, so we can run them in parallel.
	var wg sync.WaitGroup
	wg.Add(1)
	var uploadedFiles []fileset.File
	var uploadErr error
	go func() {
		defer wg.Done()
		uploadedFiles, uploadErr = files.Upload(ctx, syncOpts, false)
	}()

	var plan deployplan.Plan
	var planErr error
	if !b.DirectDeployment {
		wg.Add(1)
		go func() {
			defer wg.Done()
			plan, planErr = terraform.Plan(ctx, b.Terraform, tfDir, terraform.PlanGoal("deploy"))
		}()
	}

	wg.Wait()
	if errors.Is(uploadErr, fs.ErrPermission) {
		return permissions.ReportPossiblePermissionDenied(ctx, b, b.Config.Workspace.FilePath)
	}
	if uploadErr != nil {
		logdiag.LogError(ctx, uploadErr)
		return nil
	}
	if planErr != nil {
		logdiag.LogError(ctx, planErr)
		return nil
	}

	b.Files = uploadedFiles
	b.Plan.TerraformIsEmpty = plan.TerraformIsEmpty
	b.Plan.TerraformPlanPath = plan.TerraformPlanPath

	return nil
}
