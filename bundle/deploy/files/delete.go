package files

import (
	"context"
	"fmt"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/fatih/color"
)

type delete struct{}

func (m *delete) Name() string {
	return "files.Delete"
}

// TODO: autoapprove and tty detection for destroy. Don't allow destroy without auto-approve otherwise. Note this is a breaking change

func (m *delete) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	// Do not delete files if terraform destroy was not consented
	if !b.Plan.IsEmpty && !b.Plan.ConfirmApply {
		return nil, nil
	}

	cmdio.LogString(ctx, "\nStarting deletion of remote bundle files")
	cmdio.LogString(ctx, fmt.Sprintf("Bundle remote directory is %s", b.Config.Workspace.RootPath))

	red := color.New(color.FgRed).SprintFunc()
	if !b.AutoApprove {
		cmdio.LogString(ctx, fmt.Sprintf("\n%s and all files in it will be %s.", b.Config.Workspace.RootPath, red("deleted permanently!")))
		proceed, err := cmdio.Ask(ctx, "Proceed with deletion? [y/n]: ")
		if err != nil {
			return nil, err
		}
		if !proceed {
			return nil, nil
		}
	}

	err := b.WorkspaceClient().Workspace.Delete(ctx, workspace.Delete{
		Path:      b.Config.Workspace.RootPath,
		Recursive: true,
	})
	if err != nil {
		return nil, err
	}

	// Clean up sync snapshot file
	sync, err := getSync(ctx, b)
	if err != nil {
		return nil, err
	}
	err = sync.DestroySnapshot(ctx)
	if err != nil {
		return nil, err
	}

	cmdio.LogString(ctx, fmt.Sprintf("\nDeleted snapshot file at %s", sync.SnapshotPath()))
	cmdio.LogString(ctx, "Successfully deleted files!")
	return nil, nil
}

func Delete() bundle.Mutator {
	return &delete{}
}
