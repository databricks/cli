package bundle

import (
	"context"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/metadata"
	"github.com/databricks/cli/libs/locker"
	"github.com/databricks/cli/libs/tags"
	"github.com/databricks/cli/libs/terraform"
	"github.com/databricks/databricks-sdk-go"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type ReadOnlyBundle struct {
	b *Bundle
}

func ReadOnly(b *Bundle) ReadOnlyBundle {
	return ReadOnlyBundle{b: b}
}

func (r ReadOnlyBundle) Config() config.ReadOnlyConfig {
	return config.ReadOnly(r.b.Config)
}

func (r ReadOnlyBundle) AutoApprove() bool {
	return r.b.AutoApprove
}

func (r ReadOnlyBundle) Locker() *locker.Locker {
	return r.b.Locker
}

func (r ReadOnlyBundle) Metadata() metadata.Metadata {
	return r.b.Metadata
}

func (r ReadOnlyBundle) Plan() *terraform.Plan {
	return r.b.Plan
}

func (r ReadOnlyBundle) RootPath() string {
	return r.b.RootPath
}

func (r ReadOnlyBundle) Tagging() tags.Cloud {
	return r.b.Tagging
}

func (r ReadOnlyBundle) Terraform() *tfexec.Terraform {
	return r.b.Terraform
}

func (r ReadOnlyBundle) WorkspaceClient() *databricks.WorkspaceClient {
	return r.b.WorkspaceClient()
}

func (r ReadOnlyBundle) CacheDir(ctx context.Context, paths ...string) (string, error) {
	return r.b.CacheDir(ctx, paths...)
}

func (r ReadOnlyBundle) InternalDir(ctx context.Context) (string, error) {
	return r.b.InternalDir(ctx)
}

func (r ReadOnlyBundle) SetWorkpaceClient(w *databricks.WorkspaceClient) {
	r.b.SetWorkpaceClient(w)
}

func (r ReadOnlyBundle) GetSyncIncludePatterns(ctx context.Context) ([]string, error) {
	return r.b.GetSyncIncludePatterns(ctx)
}
