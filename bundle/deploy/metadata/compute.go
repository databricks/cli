package metadata

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/metadata"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/libs/diag"
)

// computeFunc fills the part of b.Metadata that describes the resources of one type.
// state is the deployment state after the deploy.
type computeFunc func(ctx context.Context, b *bundle.Bundle, state resourcestate.ExportedResourcesMap) diag.Diagnostics

// resourceMetadata holds the computeFunc of every resource type that the deployment
// metadata describes. They run in the canonical order of their types (the field order of
// [config.Resources]), and the first error stops the computation.
var resourceMetadata = config.NewResourceTypeRegistry[computeFunc]("deployment metadata")

type compute struct {
	state resourcestate.ExportedResourcesMap
}

// Compute returns a mutator that computes the deployment metadata. state is the deployment
// state after the deploy, which records the immutable snapshot that is actually deployed.
func Compute(state resourcestate.ExportedResourcesMap) bundle.Mutator {
	return &compute{state: state}
}

func (m *compute) Name() string {
	return "metadata.Compute"
}

func (m *compute) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	b.Metadata = metadata.Metadata{
		Version: metadata.Version,
		Config:  metadata.Config{},
	}

	// Set Git details in metadata
	b.Metadata.Config.Bundle.Git = config.Git{
		Branch:         b.Config.Bundle.Git.Branch,
		OriginURL:      b.Config.Bundle.Git.OriginURL,
		Commit:         b.Config.Bundle.Git.Commit,
		BundleRootPath: b.Config.Bundle.Git.BundleRootPath,
	}

	// Set bundle name, target, and mode
	b.Metadata.Config.Bundle.Name = b.Config.Bundle.Name
	b.Metadata.Config.Bundle.Target = b.Config.Bundle.Target
	b.Metadata.Config.Bundle.Mode = string(b.Config.Bundle.Mode)

	var diags diag.Diagnostics
	for _, computeResources := range resourceMetadata.All() {
		diags = diags.Extend(computeResources(ctx, b, m.state))
		if diags.HasError() {
			return diags
		}
	}

	// Set file upload destination of the bundle in metadata
	b.Metadata.Config.Workspace.FilePath = b.RemoteFilePath()
	// In source-linked deployment files are not copied and resources use source files, therefore we use sync path as file path in metadata
	if b.IsSourceLinkedDeployment() {
		b.Metadata.Config.Presets.SourceLinkedDeployment = true
	}

	// Set the git folder path for deployments from the workspace
	if b.WorktreeRoot != nil && strings.HasPrefix(b.WorktreeRoot.Native(), "/Workspace/") {
		b.Metadata.Extra.GitFolderPath = b.WorktreeRoot.Native()
	}

	return diags
}
