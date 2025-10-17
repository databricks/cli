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
}

// Permission level types
type (
	AlertsPermissionLevel               string
	AppPermissionLevel                  string
	ClusterPermissionLevel              string
	DashboardPermissionLevel            string
	DatabaseInstancePermissionLevel     string
	JobPermissionLevel                  string
	MlflowExperimentPermissionLevel     string
	MlflowModelPermissionLevel          string
	ModelServingEndpointPermissionLevel string
	PipelinePermissionLevel             string
	SecretScopePermissionLevel          string
	SqlWarehousePermissionLevel         string
)

// AlertPermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any alert.
type AlertPermission struct {
	Level string `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

// AppPermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any app.
type AppPermission struct {
	Level AppPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

// ClusterPermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any cluster.
type ClusterPermission struct {
	Level ClusterPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

// DashboardPermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any dashboard.
type DashboardPermission struct {
	Level DashboardPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

// DatabaseInstancePermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any database instance.
type DatabaseInstancePermission struct {
	Level DatabaseInstancePermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

// JobPermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any job.
type JobPermission struct {
	Level JobPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

// MlflowExperimentPermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any experiment.
type MlflowExperimentPermission struct {
	Level MlflowExperimentPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

// MlflowModelPermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any model.
type MlflowModelPermission struct {
	Level MlflowModelPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

// ModelServingEndpointPermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any serving endpoint.
type ModelServingEndpointPermission struct {
	Level ModelServingEndpointPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

// PipelinePermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any pipeline.
type PipelinePermission struct {
	Level PipelinePermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

// SecretScopePermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any secret scope.
// Secret scopes permissions are mapped to Secret ACLs
type SecretScopePermission struct {
	Level SecretScopePermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

// SqlWarehousePermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any SQL warehouse.
type SqlWarehousePermission struct {
	Level SqlWarehousePermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

// IPermission interface implementations

func (p Permission) GetLevel() string                { return p.Level }
func (p Permission) GetUserName() string             { return p.UserName }
func (p Permission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p Permission) GetGroupName() string            { return p.GroupName }

func (p AlertPermission) GetLevel() string                { return p.Level }
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

func (p SecretScopePermission) GetLevel() string                { return string(p.Level) }
func (p SecretScopePermission) GetUserName() string             { return p.UserName }
func (p SecretScopePermission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p SecretScopePermission) GetGroupName() string            { return p.GroupName }

func (p SqlWarehousePermission) GetLevel() string                { return string(p.Level) }
func (p SqlWarehousePermission) GetUserName() string             { return p.UserName }
func (p SqlWarehousePermission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p SqlWarehousePermission) GetGroupName() string            { return p.GroupName }
