// One field per sdk-go API module; keep in sync with the sdk-go requirements in go.mod.

package sdkclients

import (
	"context"

	accessmanagement "github.com/databricks/sdk-go/accessmanagement/v1"
	agentkit "github.com/databricks/sdk-go/agentkit/v1"
	aifunctions "github.com/databricks/sdk-go/aifunctions/v1"
	aigateway "github.com/databricks/sdk-go/aigateway/v1"
	aisearch "github.com/databricks/sdk-go/aisearch/v1"
	alertsv1 "github.com/databricks/sdk-go/alerts/v1"
	alertsv2 "github.com/databricks/sdk-go/alerts/v2"
	apps "github.com/databricks/sdk-go/apps/v1"
	authentication "github.com/databricks/sdk-go/authentication/v1"
	budgetpolicy "github.com/databricks/sdk-go/budgetpolicy/v1"
	budgets "github.com/databricks/sdk-go/budgets/v1"
	cleanrooms "github.com/databricks/sdk-go/cleanrooms/v1"
	clusterlibraries "github.com/databricks/sdk-go/clusterlibraries/v2"
	clusterpolicies "github.com/databricks/sdk-go/clusterpolicies/v2"
	clusters "github.com/databricks/sdk-go/clusters/v2"
	commandexecution "github.com/databricks/sdk-go/commandexecution/v2"
	customllms "github.com/databricks/sdk-go/customllms/v1"
	database "github.com/databricks/sdk-go/database/v1"
	dataclassification "github.com/databricks/sdk-go/dataclassification/v1"
	dataquality "github.com/databricks/sdk-go/dataquality/v1"
	disasterrecovery "github.com/databricks/sdk-go/disasterrecovery/v1"
	domains "github.com/databricks/sdk-go/domains/v1"
	environments "github.com/databricks/sdk-go/environments/v1"
	experiments "github.com/databricks/sdk-go/experiments/v1"
	features "github.com/databricks/sdk-go/features/v1"
	featurestore "github.com/databricks/sdk-go/featurestore/v1"
	files "github.com/databricks/sdk-go/files/v2"
	forecasting "github.com/databricks/sdk-go/forecasting/v1"
	genie "github.com/databricks/sdk-go/genie/v1"
	gitcredentials "github.com/databricks/sdk-go/gitcredentials/v1"
	globalinitscripts "github.com/databricks/sdk-go/globalinitscripts/v2"
	instancepools "github.com/databricks/sdk-go/instancepools/v2"
	instanceprofiles "github.com/databricks/sdk-go/instanceprofiles/v2"
	jobs "github.com/databricks/sdk-go/jobs/v2"
	keyconfigurations "github.com/databricks/sdk-go/keyconfigurations/v1"
	knowledgeassistants "github.com/databricks/sdk-go/knowledgeassistants/v1"
	lakeview "github.com/databricks/sdk-go/lakeview/v1"
	logdelivery "github.com/databricks/sdk-go/logdelivery/v1"
	marketplaces "github.com/databricks/sdk-go/marketplaces/v1"
	modelregistry "github.com/databricks/sdk-go/modelregistry/v1"
	modelserving "github.com/databricks/sdk-go/modelserving/v1"
	modelservingquery "github.com/databricks/sdk-go/modelservingquery/v1"
	networking "github.com/databricks/sdk-go/networking/v1"
	notificationdestinations "github.com/databricks/sdk-go/notificationdestinations/v1"
	oauth "github.com/databricks/sdk-go/oauth/v1"
	"github.com/databricks/sdk-go/options/client"
	pipelines "github.com/databricks/sdk-go/pipelines/v2"
	policyfamilies "github.com/databricks/sdk-go/policyfamilies/v2"
	postgres "github.com/databricks/sdk-go/postgres/v1"
	queries "github.com/databricks/sdk-go/queries/v1"
	queryhistory "github.com/databricks/sdk-go/queryhistory/v1"
	repos "github.com/databricks/sdk-go/repos/v1"
	sandbox "github.com/databricks/sdk-go/sandbox/v1"
	scim "github.com/databricks/sdk-go/scim/v1"
	secrets "github.com/databricks/sdk-go/secrets/v1"
	settings "github.com/databricks/sdk-go/settings/v2"
	sharing "github.com/databricks/sdk-go/sharing/v1"
	statementexecution "github.com/databricks/sdk-go/statementexecution/v1"
	storageconfigurations "github.com/databricks/sdk-go/storageconfigurations/v1"
	supervisoragents "github.com/databricks/sdk-go/supervisoragents/v1"
	tagassignments "github.com/databricks/sdk-go/tagassignments/v1"
	tagpolicies "github.com/databricks/sdk-go/tagpolicies/v1"
	tokenmanagement "github.com/databricks/sdk-go/tokenmanagement/v1"
	tokens "github.com/databricks/sdk-go/tokens/v1"
	ucabacpolicies "github.com/databricks/sdk-go/uc/abacpolicies/v1"
	ucartifactallowlists "github.com/databricks/sdk-go/uc/artifactallowlists/v1"
	uccatalogs "github.com/databricks/sdk-go/uc/catalogs/v1"
	ucconnections "github.com/databricks/sdk-go/uc/connections/v1"
	uccredentials "github.com/databricks/sdk-go/uc/credentials/v1"
	ucentitytagassignments "github.com/databricks/sdk-go/uc/entitytagassignments/v1"
	ucexternallineage "github.com/databricks/sdk-go/uc/externallineage/v1"
	ucexternallocations "github.com/databricks/sdk-go/uc/externallocations/v1"
	ucexternalmetadata "github.com/databricks/sdk-go/uc/externalmetadata/v1"
	ucfunctions "github.com/databricks/sdk-go/uc/functions/v1"
	ucgrants "github.com/databricks/sdk-go/uc/grants/v1"
	ucmetastores "github.com/databricks/sdk-go/uc/metastores/v1"
	uconlinetables "github.com/databricks/sdk-go/uc/onlinetables/v1"
	ucregisteredmodels "github.com/databricks/sdk-go/uc/registeredmodels/v1"
	ucresourcequotas "github.com/databricks/sdk-go/uc/resourcequotas/v1"
	ucrfa "github.com/databricks/sdk-go/uc/rfa/v1"
	ucschemas "github.com/databricks/sdk-go/uc/schemas/v1"
	ucsecrets "github.com/databricks/sdk-go/uc/secrets/v1"
	ucsystemschemas "github.com/databricks/sdk-go/uc/systemschemas/v1"
	uctables "github.com/databricks/sdk-go/uc/tables/v1"
	ucvolumes "github.com/databricks/sdk-go/uc/volumes/v1"
	ucworkspacebindings "github.com/databricks/sdk-go/uc/workspacebindings/v1"
	usagedashboards "github.com/databricks/sdk-go/usagedashboards/v1"
	vectorsearch "github.com/databricks/sdk-go/vectorsearch/v1"
	warehouses "github.com/databricks/sdk-go/warehouses/v1"
	workspaces "github.com/databricks/sdk-go/workspaces/v1"
)

// services holds one sdk-go client per API module.
type services struct {
	AccessManagement         *accessmanagement.Client
	AgentKit                 *agentkit.Client
	AiFunctions              *aifunctions.Client
	AiGateway                *aigateway.Client
	AiSearch                 *aisearch.Client
	AlertsV1                 *alertsv1.Client
	Alerts                   *alertsv2.Client
	Apps                     *apps.Client
	Authentication           *authentication.Client
	BudgetPolicy             *budgetpolicy.Client
	Budgets                  *budgets.Client
	CleanRooms               *cleanrooms.Client
	Libraries                *clusterlibraries.Client
	ClusterPolicies          *clusterpolicies.Client
	Clusters                 *clusters.Client
	CommandExecution         *commandexecution.Client
	CustomLlms               *customllms.Client
	Database                 *database.Client
	DataClassification       *dataclassification.Client
	DataQuality              *dataquality.Client
	DisasterRecovery         *disasterrecovery.Client
	Domains                  *domains.Client
	Environments             *environments.Client
	Experiments              *experiments.Client
	Features                 *features.Client
	FeatureStore             *featurestore.Client
	Files                    *files.Client
	Forecasting              *forecasting.Client
	Genie                    *genie.Client
	GitCredentials           *gitcredentials.Client
	GlobalInitScripts        *globalinitscripts.Client
	InstancePools            *instancepools.Client
	InstanceProfiles         *instanceprofiles.Client
	Jobs                     *jobs.Client
	KeyConfigurations        *keyconfigurations.Client
	KnowledgeAssistants      *knowledgeassistants.Client
	Lakeview                 *lakeview.Client
	LogDelivery              *logdelivery.Client
	Marketplaces             *marketplaces.Client
	ModelRegistry            *modelregistry.Client
	ServingEndpoints         *modelserving.Client
	ServingQuery             *modelservingquery.Client
	Networking               *networking.Client
	NotificationDestinations *notificationdestinations.Client
	OAuth                    *oauth.Client
	Pipelines                *pipelines.Client
	PolicyFamilies           *policyfamilies.Client
	Postgres                 *postgres.Client
	Queries                  *queries.Client
	QueryHistory             *queryhistory.Client
	Repos                    *repos.Client
	Sandbox                  *sandbox.Client
	Scim                     *scim.Client
	Secrets                  *secrets.Client
	Settings                 *settings.Client
	Sharing                  *sharing.Client
	StatementExecution       *statementexecution.Client
	StorageConfigurations    *storageconfigurations.Client
	SupervisorAgents         *supervisoragents.Client
	TagAssignments           *tagassignments.Client
	TagPolicies              *tagpolicies.Client
	TokenManagement          *tokenmanagement.Client
	Tokens                   *tokens.Client
	AbacPolicies             *ucabacpolicies.Client
	ArtifactAllowlists       *ucartifactallowlists.Client
	Catalogs                 *uccatalogs.Client
	Connections              *ucconnections.Client
	Credentials              *uccredentials.Client
	EntityTagAssignments     *ucentitytagassignments.Client
	ExternalLineage          *ucexternallineage.Client
	ExternalLocations        *ucexternallocations.Client
	ExternalMetadata         *ucexternalmetadata.Client
	Functions                *ucfunctions.Client
	Grants                   *ucgrants.Client
	Metastores               *ucmetastores.Client
	OnlineTables             *uconlinetables.Client
	RegisteredModels         *ucregisteredmodels.Client
	ResourceQuotas           *ucresourcequotas.Client
	Rfa                      *ucrfa.Client
	Schemas                  *ucschemas.Client
	SecretsUc                *ucsecrets.Client
	SystemSchemas            *ucsystemschemas.Client
	Tables                   *uctables.Client
	Volumes                  *ucvolumes.Client
	WorkspaceBindings        *ucworkspacebindings.Client
	UsageDashboards          *usagedashboards.Client
	VectorSearch             *vectorsearch.Client
	Warehouses               *warehouses.Client
	Workspaces               *workspaces.Client
}

func newServices(ctx context.Context, opts ...client.Option) (*services, error) {
	c := &services{}
	var err error
	if c.AccessManagement, err = accessmanagement.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.AgentKit, err = agentkit.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.AiFunctions, err = aifunctions.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.AiGateway, err = aigateway.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.AiSearch, err = aisearch.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.AlertsV1, err = alertsv1.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Alerts, err = alertsv2.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Apps, err = apps.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Authentication, err = authentication.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.BudgetPolicy, err = budgetpolicy.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Budgets, err = budgets.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.CleanRooms, err = cleanrooms.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Libraries, err = clusterlibraries.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.ClusterPolicies, err = clusterpolicies.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Clusters, err = clusters.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.CommandExecution, err = commandexecution.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.CustomLlms, err = customllms.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Database, err = database.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.DataClassification, err = dataclassification.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.DataQuality, err = dataquality.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.DisasterRecovery, err = disasterrecovery.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Domains, err = domains.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Environments, err = environments.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Experiments, err = experiments.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Features, err = features.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.FeatureStore, err = featurestore.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Files, err = files.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Forecasting, err = forecasting.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Genie, err = genie.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.GitCredentials, err = gitcredentials.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.GlobalInitScripts, err = globalinitscripts.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.InstancePools, err = instancepools.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.InstanceProfiles, err = instanceprofiles.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Jobs, err = jobs.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.KeyConfigurations, err = keyconfigurations.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.KnowledgeAssistants, err = knowledgeassistants.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Lakeview, err = lakeview.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.LogDelivery, err = logdelivery.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Marketplaces, err = marketplaces.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.ModelRegistry, err = modelregistry.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.ServingEndpoints, err = modelserving.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.ServingQuery, err = modelservingquery.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Networking, err = networking.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.NotificationDestinations, err = notificationdestinations.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.OAuth, err = oauth.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Pipelines, err = pipelines.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.PolicyFamilies, err = policyfamilies.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Postgres, err = postgres.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Queries, err = queries.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.QueryHistory, err = queryhistory.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Repos, err = repos.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Sandbox, err = sandbox.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Scim, err = scim.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Secrets, err = secrets.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Settings, err = settings.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Sharing, err = sharing.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.StatementExecution, err = statementexecution.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.StorageConfigurations, err = storageconfigurations.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.SupervisorAgents, err = supervisoragents.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.TagAssignments, err = tagassignments.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.TagPolicies, err = tagpolicies.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.TokenManagement, err = tokenmanagement.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Tokens, err = tokens.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.AbacPolicies, err = ucabacpolicies.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.ArtifactAllowlists, err = ucartifactallowlists.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Catalogs, err = uccatalogs.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Connections, err = ucconnections.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Credentials, err = uccredentials.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.EntityTagAssignments, err = ucentitytagassignments.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.ExternalLineage, err = ucexternallineage.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.ExternalLocations, err = ucexternallocations.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.ExternalMetadata, err = ucexternalmetadata.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Functions, err = ucfunctions.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Grants, err = ucgrants.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Metastores, err = ucmetastores.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.OnlineTables, err = uconlinetables.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.RegisteredModels, err = ucregisteredmodels.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.ResourceQuotas, err = ucresourcequotas.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Rfa, err = ucrfa.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Schemas, err = ucschemas.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.SecretsUc, err = ucsecrets.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.SystemSchemas, err = ucsystemschemas.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Tables, err = uctables.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Volumes, err = ucvolumes.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.WorkspaceBindings, err = ucworkspacebindings.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.UsageDashboards, err = usagedashboards.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.VectorSearch, err = vectorsearch.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Warehouses, err = warehouses.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	if c.Workspaces, err = workspaces.NewClient(ctx, opts...); err != nil {
		return nil, err
	}
	return c, nil
}
