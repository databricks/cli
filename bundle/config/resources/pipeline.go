package resources

import (
	"context"
	"net/url"
	"slices"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type Pipeline struct {
	BaseResource
	pipelines.CreatePipeline        //nolint:govet // CreatePipeline.Id and our depth-0 ID field both carry json:"id"; the depth-0 field wins
	ID                       string `json:"id,omitempty" bundle:"readonly"`

	Permissions []PipelinePermission `json:"permissions,omitempty"`

	// CascadeOnDestroy controls whether destroying the pipeline also deletes its datasets
	// Nil (unset) uses the server default, which cascades. Set to false to retain datasets
	// when the pipeline is deleted. Delete-time only: never sent on create/update.
	CascadeOnDestroy *bool `json:"cascade_on_destroy,omitempty"`
}

func (p *Pipeline) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, p)
}

func (p Pipeline) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(p)
}

func (p *Pipeline) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.Pipelines.Get(ctx, pipelines.GetPipelineRequest{
		PipelineId: id,
	})
	if err != nil {
		log.Debugf(ctx, "pipeline %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (p *Pipeline) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "pipeline",
		PluralName:    "pipelines",
		SingularTitle: "Pipeline",
		PluralTitle:   "Pipelines",
	}
}

func (*Pipeline) PermissionsSupport() PermissionsSupport {
	return PermissionsSupport{
		BundleLevels: BundlePermissionLevels{
			CanManage: "CAN_MANAGE",
			CanView:   "CAN_VIEW",
			CanRun:    "CAN_RUN",
		},
		RequestObjectType:  "pipelines",
		ResponseObjectType: "pipelines",
		ObjectIDField:      "id",
		HasIsOwner:         true,
	}
}

func (*Pipeline) WorkspaceURL() (workspaceurls.Pattern, bool) {
	return workspaceurls.Pattern{Path: "pipelines/%s"}, true
}

func (p *Pipeline) InitializeURL(baseURL url.URL) {
	if p.ID == "" {
		return
	}
	p.URL = workspaceURL(p, baseURL, p.ID)
}

func (p *Pipeline) GetName() string {
	return p.Name
}

func (*Pipeline) RunAsSupport() RunAsSupport {
	return RunAsApplied
}

func (p *Pipeline) ApplyRunAs(runAs *jobs.JobRunAs) {
	if p.RunAs != nil {
		return
	}
	p.RunAs = &pipelines.RunAs{
		GroupName:            runAs.GroupName,
		ServicePrincipalName: runAs.ServicePrincipalName,
		UserName:             runAs.UserName,
	}
}

// Legacy behavior of run_as for SDP pipelines. Available under the experimental.use_run_as_legacy flag.
// Only available to unblock customers stuck due to breaking changes in https://github.com/databricks/cli/pull/1233
func (p *Pipeline) ApplyLegacyRunAs(runAs *jobs.JobRunAs, currentUser string) {
	// If user deploying the bundle and the one defined in run_as are the same
	// Do not add IS_OWNER permission. Current user is implied to be an owner in this case.
	// Otherwise, it will fail due to this bug https://github.com/databricks/terraform-provider-databricks/issues/2407
	if runAs.UserName == currentUser || runAs.ServicePrincipalName == currentUser {
		return
	}

	p.Permissions = slices.DeleteFunc(p.Permissions, func(perm PipelinePermission) bool {
		return (runAs.ServicePrincipalName != "" && perm.ServicePrincipalName == runAs.ServicePrincipalName) ||
			(runAs.UserName != "" && perm.UserName == runAs.UserName)
	})
	p.Permissions = append(p.Permissions, PipelinePermission{
		Level:                pipelines.PipelinePermissionLevelIsOwner,
		ServicePrincipalName: runAs.ServicePrincipalName,
		UserName:             runAs.UserName,
	})
}

func (p *Pipeline) UCNames() []UCName {
	// "schema" and "target" have the same semantics in the SDP API but are mutually
	// exclusive i.e. only one can be set at a time.
	return []UCName{ucCatalogAndSchemas(&p.Catalog, &p.Schema, &p.Target)}
}

func (p *Pipeline) ApplyPresets(t *PresetSettings) {
	p.Name = t.NamePrefix + p.Name
	if t.PipelinesDevelopment {
		//nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field
		p.Development = true
	}
	if t.TriggerPauseStatus == TriggerPauseStatusPaused {
		//nolint:staticcheck // SA1019: pipeline continuous is deprecated in the SDK but remains a supported bundle config field
		p.Continuous = false
	}

	if len(t.Tags) > 0 {
		if p.Tags == nil {
			// Note: only create this map if tags is not empty, to avoid inserting "tags: {}" entry in the config
			p.Tags = make(map[string]string, len(t.Tags))
		}
		for _, tag := range t.Tags {
			if p.Tags[tag.Key] == "" {
				p.Tags[tag.Key] = tag.Value
			}
		}
	}
}

func (p *Pipeline) FieldDefaults(DefaultsContext) []FieldDefault {
	return []FieldDefault{
		// Same as terraform
		// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/pipelines/resource_pipeline.go#L253
		{"edition", "ADVANCED"},
		{"channel", "CURRENT"},
	}
}

func (p *Pipeline) ValidateProductionMode() diag.Diagnostics {
	//nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field
	if p.Development {
		return diag.Errorf("target with 'mode: production' cannot include a pipeline with 'development: true'")
	}
	return nil
}

func (p *Pipeline) LocalPaths() []LocalPath {
	libraries := dyn.NewPattern(dyn.Key("libraries"), dyn.AnyIndex())
	paths := []LocalPath{
		{
			Pattern: libraries.Append(dyn.Key("notebook"), dyn.Key("path")),
			Mode:    TranslateModeNotebook,
		},
		{
			Pattern: libraries.Append(dyn.Key("file"), dyn.Key("path")),
			Mode:    TranslateModeFile,
		},
		{
			Pattern: libraries.Append(dyn.Key("glob"), dyn.Key("include")),
			Mode:    TranslateModeGlob,
		},
		{
			Pattern: dyn.NewPattern(dyn.Key("root_path")),
			Mode:    TranslateModeDirectory,
		},
		{
			Pattern: dyn.NewPattern(dyn.Key("testing"), dyn.Key("sources"), dyn.AnyIndex(), dyn.Key("glob"), dyn.Key("include")),
			Mode:    TranslateModeGlob,
		},
	}

	// Test dependencies take the same forms as the pipeline's environment dependencies.
	dependencies := []dyn.Pattern{
		dyn.NewPattern(dyn.Key("environment"), dyn.Key("dependencies"), dyn.AnyIndex()),
		dyn.NewPattern(dyn.Key("testing"), dyn.Key("environment"), dyn.Key("dependencies"), dyn.AnyIndex()),
	}
	for _, pattern := range dependencies {
		paths = append(paths, LocalPath{
			Pattern:              pattern,
			Mode:                 TranslateModeLocalRelativeWithPrefix,
			Filter:               OnlyLocalLibraries,
			AllowOutsideSyncRoot: true,
		})
	}
	for _, pattern := range dependencies {
		paths = append(paths, LocalPath{
			Pattern:              pattern,
			Mode:                 TranslateModeEnvironmentPipFlag,
			Filter:               OnlyPipFlagPaths,
			AllowOutsideSyncRoot: true,
		})
	}
	return paths
}
