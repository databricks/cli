package generate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	bundleconfigresources "github.com/databricks/cli/bundle/config/resources"
	bundlegenerate "github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/cmd/bundle/deployment"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/template"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

type generateEnvironment struct {
	hadExistingBundle bool
}

type genericGenerateSpec struct {
	commandName       string
	resourceGroup     string
	fileExtension     string
	lookupFlag        string
	lookupDescription string
	shortDescription  string
	fetch             func(context.Context, *databricks.WorkspaceClient, string) (any, error)
	convert           func(any) (dyn.Value, error)
}

func initGenerateContext(cmd *cobra.Command) context.Context {
	ctx := cmd.Context()
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
		cmd.SetContext(ctx)
	}
	return ctx
}

func ensureGenerateBundle(cmd *cobra.Command, requireExistingErr error) (*generateEnvironment, error) {
	ctx := cmd.Context()
	b := root.TryConfigureBundle(cmd)
	if logdiag.HasError(ctx) {
		return nil, root.ErrAlreadyPrinted
	}

	if b != nil {
		return &generateEnvironment{hadExistingBundle: true}, nil
	}

	if requireExistingErr != nil {
		return nil, requireExistingErr
	}

	initBare, err := cmd.Flags().GetBool("init-bare")
	if err != nil {
		return nil, err
	}
	if !initBare {
		return nil, errors.New("no databricks.yml found in this directory or any parent. Run `databricks bundle init default-bare` first or re-run this command with --init-bare")
	}

	err = initializeBareBundle(ctx)
	if err != nil {
		return nil, err
	}

	b = root.MustConfigureBundle(cmd)
	if b == nil || logdiag.HasError(cmd.Context()) {
		return nil, root.ErrAlreadyPrinted
	}

	return &generateEnvironment{hadExistingBundle: false}, nil
}

func initializeBareBundle(ctx context.Context) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	projectName := textutil.NormalizeString(filepath.Base(cwd))
	if projectName == "" {
		projectName = "my_bundle"
	}

	configFile, err := os.CreateTemp("", "databricks-bundle-generate-init-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(configFile.Name())

	_, err = configFile.WriteString(fmt.Sprintf("{\"project_name\":%q}", projectName))
	if err != nil {
		configFile.Close()
		return err
	}
	err = configFile.Close()
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, "No databricks.yml found. Initializing a bare bundle skeleton.")

	r := template.Resolver{
		TemplatePathOrUrl: string(template.DefaultBare),
		ConfigFile:        configFile.Name(),
		OutputDir:         cwd,
	}

	tmpl, err := r.Resolve(ctx)
	if err != nil {
		return err
	}
	defer tmpl.Reader.Cleanup(ctx)

	err = tmpl.Writer.Materialize(ctx, tmpl.Reader)
	if err != nil {
		return err
	}
	tmpl.Writer.LogTelemetry(ctx)
	return nil
}

func defaultGeneratedKey(cmd *cobra.Command, lookup string, response any) string {
	key := cmd.Flag("key").Value.String()
	if key != "" {
		return key
	}

	name := bestEffortResourceName(response, lookup)
	key = textutil.NormalizeString(name)
	if key != "" {
		return key
	}

	return "generated_resource"
}

func bestEffortResourceName(response any, fallback string) string {
	data, err := json.Marshal(response)
	if err != nil {
		return fallback
	}

	var values map[string]any
	err = json.Unmarshal(data, &values)
	if err != nil {
		return fallback
	}

	for _, key := range []string{"display_name", "name", "cluster_name", "full_name", "table_name"} {
		value, ok := values[key]
		if !ok {
			continue
		}

		s, ok := value.(string)
		if ok && s != "" {
			return s
		}
	}

	return fallback
}

// classifySecretScopePrincipal resolves principal into one of user / group /
// service-principal by querying SCIM. Secret-scope ACLs don't carry a type
// tag on the principal, and any of the three kinds can legitimately contain
// `@`, numeric IDs, or UUIDs, so plain string heuristics misclassify
// legitimate inputs. If SCIM lookups fail or find nothing, fall back to a
// best-effort heuristic.
func classifySecretScopePrincipal(ctx context.Context, w *databricks.WorkspaceClient, principal string, permission workspace.AclPermission) bundleconfigresources.SecretScopePermission {
	entry := bundleconfigresources.SecretScopePermission{
		Level: bundleconfigresources.SecretScopePermissionLevel(strings.ToUpper(permission.String())),
	}

	sps, err := w.ServicePrincipals.ListAll(ctx, iam.ListServicePrincipalsRequest{
		Filter: fmt.Sprintf(`applicationId eq "%s"`, principal),
	})
	if err == nil && len(sps) > 0 {
		entry.ServicePrincipalName = principal
		return entry
	}

	users, err := w.Users.ListAll(ctx, iam.ListUsersRequest{
		Filter: fmt.Sprintf(`userName eq "%s"`, principal),
	})
	if err == nil && len(users) > 0 {
		entry.UserName = principal
		return entry
	}

	groups, err := w.Groups.ListAll(ctx, iam.ListGroupsRequest{
		Filter: fmt.Sprintf(`displayName eq "%s"`, principal),
	})
	if err == nil && len(groups) > 0 {
		entry.GroupName = principal
		return entry
	}

	switch {
	case strings.Contains(principal, "@"):
		entry.UserName = principal
	case len(principal) == 36 && strings.Count(principal, "-") == 4:
		entry.ServicePrincipalName = principal
	default:
		entry.GroupName = principal
	}
	return entry
}

func convertSecretScopeToValue(scope any) (dyn.Value, error) {
	return bundlegenerate.ConvertResourceToValue(scope, nil, nil)
}

func fetchSecretScope(ctx context.Context, w *databricks.WorkspaceClient, name string) (any, error) {
	scopes, err := w.Secrets.ListScopesAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, scope := range scopes {
		if scope.Name != name {
			continue
		}

		resource := bundleconfigresources.SecretScope{
			Name:             scope.Name,
			BackendType:      scope.BackendType,
			KeyvaultMetadata: scope.KeyvaultMetadata,
		}

		acls := w.Secrets.ListAcls(ctx, workspace.ListAclsRequest{Scope: name})
		for acls.HasNext(ctx) {
			acl, err := acls.Next(ctx)
			if err != nil {
				return nil, err
			}
			resource.Permissions = append(resource.Permissions, classifySecretScopePrincipal(ctx, w, acl.Principal, acl.Permission))
		}

		return resource, nil
	}

	return nil, fmt.Errorf("secret scope %q not found", name)
}

func newGenericGenerateCommand(spec genericGenerateSpec) *cobra.Command {
	var configDir string
	var lookup string
	var force bool
	var bind bool

	cmd := &cobra.Command{
		Use:     spec.commandName,
		Short:   spec.shortDescription,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := initGenerateContext(cmd)

			var requireExistingErr error
			if bind {
				requireExistingErr = errors.New("--bind requires an existing bundle. Re-run this command from a bundle directory or omit --bind")
			}
			_, err := ensureGenerateBundle(cmd, requireExistingErr)
			if err != nil {
				return err
			}

			b := root.MustConfigureBundle(cmd)
			if b == nil || logdiag.HasError(cmd.Context()) {
				return root.ErrAlreadyPrinted
			}

			response, err := spec.fetch(ctx, b.WorkspaceClient(ctx), lookup)
			if err != nil {
				return err
			}

			value, err := spec.convert(response)
			if err != nil {
				return err
			}

			resourceKey := defaultGeneratedKey(cmd, lookup, response)
			filename := filepath.Join(configDir, resourceKey+"."+spec.fileExtension+".yml")

			result := map[string]dyn.Value{
				"resources": dyn.V(map[string]dyn.Value{
					spec.resourceGroup: dyn.V(map[string]dyn.Value{
						resourceKey: value,
					}),
				}),
			}

			err = yamlsaver.NewSaver().SaveAsYAML(result, filename, force)
			if err != nil {
				return err
			}

			cmdio.LogString(ctx, fmt.Sprintf("%s configuration successfully saved to %s", spec.shortDescription, filepath.ToSlash(filename)))

			if bind {
				return deployment.BindResource(cmd, resourceKey, lookup, true, false, true)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&lookup, spec.lookupFlag, "", spec.lookupDescription)
	err := cmd.MarkFlagRequired(spec.lookupFlag)
	if err != nil {
		panic(err)
	}

	cmd.Flags().StringVarP(&configDir, "config-dir", "d", "resources", `directory to write the configuration to`)
	cmd.Flags().BoolVarP(&force, "force", "f", false, `force overwrite existing files in the output directory`)
	cmd.Flags().BoolVarP(&bind, "bind", "b", false, `automatically bind the generated resource to the existing resource`)
	if err := cmd.Flags().MarkHidden("bind"); err != nil {
		panic(err)
	}
	return cmd
}

func genericGenerateSpecs() []genericGenerateSpec {
	return []genericGenerateSpec{
		{
			commandName:       "catalog",
			resourceGroup:     "catalogs",
			fileExtension:     "catalog",
			lookupFlag:        "existing-catalog-name",
			lookupDescription: `catalog name to generate config for`,
			shortDescription:  "Generate bundle configuration for a catalog",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.Catalogs.GetByName(ctx, lookup)
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.Catalog](response, nil, nil)
			},
		},
		{
			commandName:       "schema",
			resourceGroup:     "schemas",
			fileExtension:     "schema",
			lookupFlag:        "existing-schema-full-name",
			lookupDescription: `schema full name to generate config for`,
			shortDescription:  "Generate bundle configuration for a schema",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.Schemas.GetByFullName(ctx, lookup)
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.Schema](response, nil, nil)
			},
		},
		{
			commandName:       "external-location",
			resourceGroup:     "external_locations",
			fileExtension:     "external_location",
			lookupFlag:        "existing-external-location-name",
			lookupDescription: `external location name to generate config for`,
			shortDescription:  "Generate bundle configuration for an external location",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.ExternalLocations.GetByName(ctx, lookup)
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.ExternalLocation](response, nil, nil)
			},
		},
		{
			commandName:       "volume",
			resourceGroup:     "volumes",
			fileExtension:     "volume",
			lookupFlag:        "existing-volume-name",
			lookupDescription: `volume full name to generate config for`,
			shortDescription:  "Generate bundle configuration for a volume",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.Volumes.Read(ctx, catalog.ReadVolumeRequest{Name: lookup})
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.Volume](response, nil, nil)
			},
		},
		{
			commandName:       "cluster",
			resourceGroup:     "clusters",
			fileExtension:     "cluster",
			lookupFlag:        "existing-cluster-id",
			lookupDescription: `cluster ID to generate config for`,
			shortDescription:  "Generate bundle configuration for a cluster",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.Clusters.GetByClusterId(ctx, lookup)
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.Cluster](response, nil, nil)
			},
		},
		{
			commandName:       "sql-warehouse",
			resourceGroup:     "sql_warehouses",
			fileExtension:     "sql_warehouse",
			lookupFlag:        "existing-sql-warehouse-id",
			lookupDescription: `SQL warehouse ID to generate config for`,
			shortDescription:  "Generate bundle configuration for a SQL warehouse",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.Warehouses.GetById(ctx, lookup)
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.SqlWarehouse](response, nil, nil)
			},
		},
		{
			commandName:       "model",
			resourceGroup:     "models",
			fileExtension:     "model",
			lookupFlag:        "existing-model-name",
			lookupDescription: `model name to generate config for`,
			shortDescription:  "Generate bundle configuration for a model",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.ModelRegistry.GetModel(ctx, ml.GetModelRequest{Name: lookup})
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.MlflowModel](response, nil, nil)
			},
		},
		{
			commandName:       "experiment",
			resourceGroup:     "experiments",
			fileExtension:     "experiment",
			lookupFlag:        "existing-experiment-id",
			lookupDescription: `experiment ID to generate config for`,
			shortDescription:  "Generate bundle configuration for an experiment",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.Experiments.GetExperiment(ctx, ml.GetExperimentRequest{ExperimentId: lookup})
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.MlflowExperiment](response, nil, nil)
			},
		},
		{
			commandName:       "model-serving-endpoint",
			resourceGroup:     "model_serving_endpoints",
			fileExtension:     "model_serving_endpoint",
			lookupFlag:        "existing-model-serving-endpoint-name",
			lookupDescription: `model serving endpoint name to generate config for`,
			shortDescription:  "Generate bundle configuration for a model serving endpoint",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.ServingEndpoints.Get(ctx, serving.GetServingEndpointRequest{Name: lookup})
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.ModelServingEndpoint](response, nil, nil)
			},
		},
		{
			commandName:       "registered-model",
			resourceGroup:     "registered_models",
			fileExtension:     "registered_model",
			lookupFlag:        "existing-registered-model-full-name",
			lookupDescription: `registered model full name to generate config for`,
			shortDescription:  "Generate bundle configuration for a registered model",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.RegisteredModels.Get(ctx, catalog.GetRegisteredModelRequest{FullName: lookup})
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.RegisteredModel](response, nil, nil)
			},
		},
		{
			commandName:       "quality-monitor",
			resourceGroup:     "quality_monitors",
			fileExtension:     "quality_monitor",
			lookupFlag:        "existing-quality-monitor-table-name",
			lookupDescription: `table name of the quality monitor to generate config for`,
			shortDescription:  "Generate bundle configuration for a quality monitor",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.QualityMonitors.Get(ctx, catalog.GetQualityMonitorRequest{TableName: lookup})
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.QualityMonitor](response, nil, nil)
			},
		},
		{
			commandName:       "secret-scope",
			resourceGroup:     "secret_scopes",
			fileExtension:     "secret_scope",
			lookupFlag:        "existing-secret-scope-name",
			lookupDescription: `secret scope name to generate config for`,
			shortDescription:  "Generate bundle configuration for a secret scope",
			fetch:             fetchSecretScope,
			convert:           convertSecretScopeToValue,
		},
		{
			commandName:       "database-instance",
			resourceGroup:     "database_instances",
			fileExtension:     "database_instance",
			lookupFlag:        "existing-database-instance-name",
			lookupDescription: `database instance name to generate config for`,
			shortDescription:  "Generate bundle configuration for a database instance",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.Database.GetDatabaseInstance(ctx, database.GetDatabaseInstanceRequest{Name: lookup})
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.DatabaseInstance](response, nil, nil)
			},
		},
		{
			commandName:       "database-catalog",
			resourceGroup:     "database_catalogs",
			fileExtension:     "database_catalog",
			lookupFlag:        "existing-database-catalog-name",
			lookupDescription: `database catalog name to generate config for`,
			shortDescription:  "Generate bundle configuration for a database catalog",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.Database.GetDatabaseCatalog(ctx, database.GetDatabaseCatalogRequest{Name: lookup})
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.DatabaseCatalog](response, nil, nil)
			},
		},
		{
			commandName:       "synced-database-table",
			resourceGroup:     "synced_database_tables",
			fileExtension:     "synced_database_table",
			lookupFlag:        "existing-synced-database-table-name",
			lookupDescription: `synced database table name to generate config for`,
			shortDescription:  "Generate bundle configuration for a synced database table",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.Database.GetSyncedDatabaseTable(ctx, database.GetSyncedDatabaseTableRequest{Name: lookup})
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.SyncedDatabaseTable](response, nil, nil)
			},
		},
		{
			commandName:       "postgres-project",
			resourceGroup:     "postgres_projects",
			fileExtension:     "postgres_project",
			lookupFlag:        "existing-postgres-project-name",
			lookupDescription: `postgres project name to generate config for`,
			shortDescription:  "Generate bundle configuration for a postgres project",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.Postgres.GetProject(ctx, postgres.GetProjectRequest{Name: lookup})
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.PostgresProject](response, nil, nil)
			},
		},
		{
			commandName:       "postgres-branch",
			resourceGroup:     "postgres_branches",
			fileExtension:     "postgres_branch",
			lookupFlag:        "existing-postgres-branch-name",
			lookupDescription: `postgres branch name to generate config for`,
			shortDescription:  "Generate bundle configuration for a postgres branch",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.Postgres.GetBranch(ctx, postgres.GetBranchRequest{Name: lookup})
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.PostgresBranch](response, nil, nil)
			},
		},
		{
			commandName:       "postgres-endpoint",
			resourceGroup:     "postgres_endpoints",
			fileExtension:     "postgres_endpoint",
			lookupFlag:        "existing-postgres-endpoint-name",
			lookupDescription: `postgres endpoint name to generate config for`,
			shortDescription:  "Generate bundle configuration for a postgres endpoint",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.Postgres.GetEndpoint(ctx, postgres.GetEndpointRequest{Name: lookup})
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.PostgresEndpoint](response, nil, nil)
			},
		},
		{
			commandName:       "vector-search-endpoint",
			resourceGroup:     "vector_search_endpoints",
			fileExtension:     "vector_search_endpoint",
			lookupFlag:        "existing-vector-search-endpoint-name",
			lookupDescription: `vector search endpoint name to generate config for`,
			shortDescription:  "Generate bundle configuration for a vector search endpoint",
			fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
				return w.VectorSearchEndpoints.GetEndpoint(ctx, vectorsearch.GetEndpointRequest{EndpointName: lookup})
			},
			convert: func(response any) (dyn.Value, error) {
				return bundlegenerate.ConvertResponseToValue[bundleconfigresources.VectorSearchEndpoint](response, nil, nil)
			},
		},
	}
}

func NewGenericGenerateCommands() []*cobra.Command {
	specs := genericGenerateSpecs()
	cmds := make([]*cobra.Command, 0, len(specs))
	for _, spec := range specs {
		cmds = append(cmds, newGenericGenerateCommand(spec))
	}
	return cmds
}
