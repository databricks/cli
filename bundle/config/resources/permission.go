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

type (
	JobPermissionLevel      string
	PipelinePermissionLevel string
)

// JobPermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any job.
type JobPermission struct {
	Level JobPermissionLevel `json:"level"`

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

func (p JobPermission) GetLevel() string                { return string(p.Level) }
func (p JobPermission) GetUserName() string             { return p.UserName }
func (p JobPermission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p JobPermission) GetGroupName() string            { return p.GroupName }

func (p PipelinePermission) GetLevel() string                { return string(p.Level) }
func (p PipelinePermission) GetUserName() string             { return p.UserName }
func (p PipelinePermission) GetServicePrincipalName() string { return p.ServicePrincipalName }
func (p PipelinePermission) GetGroupName() string            { return p.GroupName }
