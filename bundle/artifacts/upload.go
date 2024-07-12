package artifacts

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
)

func UploadAll() bundle.Mutator {
	return &all{
		name: "Upload",
		fn:   uploadArtifactByName,
	}
}

func CleanUp() bundle.Mutator {
	return &cleanUp{}
}

type upload struct {
	name string
}

func uploadArtifactByName(name string) (bundle.Mutator, error) {
	return &upload{name}, nil
}

func (m *upload) Name() string {
	return fmt.Sprintf("artifacts.Upload(%s)", m.name)
}

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return diag.Errorf("artifact doesn't exist: %s", m.name)
	}

	if len(artifact.Files) == 0 {
		return diag.Errorf("artifact source is not configured: %s", m.name)
	}

	return bundle.Apply(ctx, b, getUploadMutator(artifact.Type, m.name))
}

type cleanUp struct{}

func (m *cleanUp) Name() string {
	return "artifacts.CleanUp"
}

func (m *cleanUp) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	uploadPath, err := getUploadBasePath(b)
	if err != nil {
		return diag.FromErr(err)
	}

	client, err := getFilerForArtifacts(b.WorkspaceClient(), uploadPath)
	if err != nil {
		return diag.FromErr(err)
	}

	// We interntionally ignore the error because it is not critical to the deployment
	client.Delete(ctx, "", filer.DeleteRecursively)

	err = client.Mkdir(ctx, "")
	if err != nil {
		return diag.Errorf("unable to create directory for %s: %v", uploadPath, err)
	}

	return nil
}
