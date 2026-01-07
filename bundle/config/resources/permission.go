package resources

import "fmt"

// Permission holds the permission level setting for a single principal.
type Permission struct {
	Level string `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

func (p Permission) String() string {
	if p.UserName != "" {
		return fmt.Sprintf("level: %s, user_name: %s", p.Level, p.UserName)
	}

	if p.ServicePrincipalName != "" {
		return fmt.Sprintf("level: %s, service_principal_name: %s", p.Level, p.ServicePrincipalName)
	}

	if p.GroupName != "" {
		return fmt.Sprintf("level: %s, group_name: %s", p.Level, p.GroupName)
	}

	return "level: " + p.Level
}

type IPermission interface {
	GetLevel() string
	GetUserName() string
	GetServicePrincipalName() string
	GetGroupName() string
	GetAPIRequestObjectType() string
}

// Permission level types
type (
	AlertPermissionLevel                string
	AppPermissionLevel                  string
	ClusterPermissionLevel              string
	DashboardPermissionLevel            string
	DatabaseInstancePermissionLevel     string
	GenieSpacePermissionLevel           string
	JobPermissionLevel                  string
	MlflowExperimentPermissionLevel     string
	MlflowModelPermissionLevel          string
	ModelServingEndpointPermissionLevel string
	PipelinePermissionLevel             string
	SqlWarehousePermissionLevel         string
)

func (l AlertPermissionLevel) Values() []string {
	return []string{
		"CAN_EDIT",
		"CAN_MANAGE",
		"CAN_READ",
		"CAN_RUN",
	}
}

type AlertPermission struct {
	Level AlertPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type AppPermission struct {
	Level AppPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type ClusterPermission struct {
	Level ClusterPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type DashboardPermission struct {
	Level DashboardPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type DatabaseInstancePermission struct {
	Level DatabaseInstancePermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type GenieSpacePermission struct {
	Level GenieSpacePermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type JobPermission struct {
	Level JobPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type MlflowExperimentPermission struct {
	Level MlflowExperimentPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type MlflowModelPermission struct {
	Level MlflowModelPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type ModelServingEndpointPermission struct {
	Level ModelServingEndpointPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type PipelinePermission struct {
	Level PipelinePermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type SqlWarehousePermission struct {
	Level SqlWarehousePermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

// GetAPIRequestObjectType is used by direct to construct a request to permissions API:
// https://github.com/databricks/terraform-provider-databricks/blob/430902d/permissions/permission_definitions.go#L775C24-L775C32
func (p AlertPermission) GetAPIRequestObjectType() string            { return "/alertsv2/" }
func (p AppPermission) GetAPIRequestObjectType() string              { return "/apps/" }
func (p ClusterPermission) GetAPIRequestObjectType() string          { return "/clusters/" }
func (p DashboardPermission) GetAPIRequestObjectType() string        { return "/dashboards/" }
func (p DatabaseInstancePermission) GetAPIRequestObjectType() string { return "/database-instances/" }
func (p GenieSpacePermission) GetAPIRequestObjectType() string       { return "/genie/spaces/" }
func (p JobPermission) GetAPIRequestObjectType() string              { return "/jobs/" }
func (p MlflowExperimentPermission) GetAPIRequestObjectType() string { return "/experiments/" }
func (p MlflowModelPermission) GetAPIRequestObjectType() string      { return "/registered-models/" }
func (p ModelServingEndpointPermission) GetAPIRequestObjectType() string {
	return "/serving-endpoints/"
}
func (p PipelinePermission) GetAPIRequestObjectType() string     { return "/pipelines/" }
func (p SqlWarehousePermission) GetAPIRequestObjectType() string { return "/sql/warehouses/" }

// IPermission interface implementations boilerplate

func (p AlertPermission) GetLevel() string                { return string(p.Level) }
func (p AlertPermission) GetUserName() string             { return p.UserName }
func (p AlertPermission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p AlertPermission) GetGroupName() string            { return p.GroupName }

func (p AppPermission) GetLevel() string                { return string(p.Level) }
func (p AppPermission) GetUserName() string             { return p.UserName }
func (p AppPermission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p AppPermission) GetGroupName() string            { return p.GroupName }

func (p ClusterPermission) GetLevel() string                { return string(p.Level) }
func (p ClusterPermission) GetUserName() string             { return p.UserName }
func (p ClusterPermission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p ClusterPermission) GetGroupName() string            { return p.GroupName }

func (p DashboardPermission) GetLevel() string                { return string(p.Level) }
func (p DashboardPermission) GetUserName() string             { return p.UserName }
func (p DashboardPermission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p DashboardPermission) GetGroupName() string            { return p.GroupName }

func (p DatabaseInstancePermission) GetLevel() string                { return string(p.Level) }
func (p DatabaseInstancePermission) GetUserName() string             { return p.UserName }
func (p DatabaseInstancePermission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p DatabaseInstancePermission) GetGroupName() string            { return p.GroupName }

func (p GenieSpacePermission) GetLevel() string                { return string(p.Level) }
func (p GenieSpacePermission) GetUserName() string             { return p.UserName }
func (p GenieSpacePermission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p GenieSpacePermission) GetGroupName() string            { return p.GroupName }

func (p JobPermission) GetLevel() string                { return string(p.Level) }
func (p JobPermission) GetUserName() string             { return p.UserName }
func (p JobPermission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p JobPermission) GetGroupName() string            { return p.GroupName }

func (p MlflowExperimentPermission) GetLevel() string                { return string(p.Level) }
func (p MlflowExperimentPermission) GetUserName() string             { return p.UserName }
func (p MlflowExperimentPermission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p MlflowExperimentPermission) GetGroupName() string            { return p.GroupName }

func (p MlflowModelPermission) GetLevel() string                { return string(p.Level) }
func (p MlflowModelPermission) GetUserName() string             { return p.UserName }
func (p MlflowModelPermission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p MlflowModelPermission) GetGroupName() string            { return p.GroupName }

func (p ModelServingEndpointPermission) GetLevel() string    { return string(p.Level) }
func (p ModelServingEndpointPermission) GetUserName() string { return p.UserName }
func (p ModelServingEndpointPermission) GetServicePrincipalName() string {
	return p.ServicePrincipalName
}
func (p ModelServingEndpointPermission) GetGroupName() string { return p.GroupName }

func (p PipelinePermission) GetLevel() string                { return string(p.Level) }
func (p PipelinePermission) GetUserName() string             { return p.UserName }
func (p PipelinePermission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p PipelinePermission) GetGroupName() string            { return p.GroupName }

func (p SqlWarehousePermission) GetLevel() string                { return string(p.Level) }
func (p SqlWarehousePermission) GetUserName() string             { return p.UserName }
func (p SqlWarehousePermission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p SqlWarehousePermission) GetGroupName() string            { return p.GroupName }
