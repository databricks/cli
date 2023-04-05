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

func (m *delete) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	// Do not delete files if terraform destroy was not consented
	if !b.Plan.IsEmpty && !b.Plan.ConfirmApply {
		return nil, nil
	}

	// interface to io with the user
	logger, ok := cmdio.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no logger found")
	}
	red := color.New(color.FgRed).SprintFunc()
	proceed, err := logger.Ask(fmt.Sprintf("\nDirectory %s and all files in it will be %s Proceed?: ", b.Config.Workspace.Root, red("deleted permanently!")))
	if err != nil {
		return nil, err
	}
	if !proceed {
		return nil, nil
	}

	err = b.WorkspaceClient().Workspace.Delete(ctx, workspace.Delete{
		Path:      b.Config.Workspace.Root,
		Recursive: true,
	})
	if err != nil {
		return nil, err
	}

	fmt.Println("Successfully deleted files!")
	return nil, nil
}

func Delete() bundle.Mutator {
	return &delete{}
}
