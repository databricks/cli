package files

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/fatih/color"
)

type delete struct{}

func (m *delete) Name() string {
	return "files.Delete"
}

func (m *delete) Apply(ctx context.Context, b *bundle.Bundle) error {
	// Do not delete files if terraform destroy was not consented
	if !b.Plan.IsEmpty && !b.Plan.ConfirmApply {
		return nil
	}

	cmdio.LogString(ctx, fmt.Sprintf("Deleting remote bundle directory %s", b.Config.Workspace.RootPath))

	red := color.New(color.FgRed).SprintFunc()
	if !b.AutoApprove {
		proceed, err := cmdio.AskYesOrNo(ctx, fmt.Sprintf("\n%s and any files in it will be permanently %s. Proceed?", b.Config.Workspace.RootPath, red("deleted")))
		if err != nil {
			return err
		}
		if !proceed {
			return nil
		}
	}

	err := b.WorkspaceClient().Workspace.Delete(ctx, workspace.Delete{
		Path:      b.Config.Workspace.RootPath,
		Recursive: true,
	})
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, "Deleted remote bundle directory")

	// Clean up sync snapshot file
	sync, err := getSync(ctx, b)
	if err != nil {
		return err
	}
	err = sync.DestroySnapshot(ctx)
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, fmt.Sprintf("Deleted local snapshot file at %s", sync.SnapshotPath()))
	return nil
}

func Delete() bundle.Mutator {
	return &delete{}
}
