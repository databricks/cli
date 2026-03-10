package resources

import (
	"fmt"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/iam"
)

// Permission holds the permission level setting for a single principal.
type Permission[L ~string] struct {
	Level L `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

// ToAccessControlRequest converts to the SDK type used by the permissions API.
func (p Permission[L]) ToAccessControlRequest() iam.AccessControlRequest {
	return iam.AccessControlRequest{
		PermissionLevel:      iam.PermissionLevel(p.Level),
		UserName:             p.UserName,
		ServicePrincipalName: p.ServicePrincipalName,
		GroupName:            p.GroupName,
	}
}

func (p Permission[L]) String() string {
	if p.UserName != "" {
		return fmt.Sprintf("level: %s, user_name: %s", p.Level, p.UserName)
	}

	if p.ServicePrincipalName != "" {
		return fmt.Sprintf("level: %s, service_principal_name: %s", p.Level, p.ServicePrincipalName)
	}

	if p.GroupName != "" {
		return fmt.Sprintf("level: %s, group_name: %s", p.Level, p.GroupName)
	}

	return "level: " + string(p.Level)
}

// PermissionOrder defines the hierarchy of permission levels.
// Higher numbers mean more permissive access.
// Based on https://docs.databricks.com/aws/en/security/auth/access-control
var PermissionOrder = map[string]int{
	"":                               -1,
	"CAN_VIEW":                       2,
	"CAN_READ":                       3,
	"CAN_VIEW_METADATA":              4,
	"CAN_RUN":                        5,
	"CAN_QUERY":                      6,
	"CAN_USE":                        7,
	"CAN_EDIT":                       8,
	"CAN_EDIT_METADATA":              9,
	"CAN_CREATE":                     10,
	"CAN_ATTACH_TO":                  11,
	"CAN_RESTART":                    12,
	"CAN_MONITOR":                    13,
	"CAN_MANAGE_RUN":                 14,
	"CAN_MANAGE_STAGING_VERSIONS":    15,
	"CAN_MANAGE_PRODUCTION_VERSIONS": 16,
	"CAN_MANAGE":                     17,
	"IS_OWNER":                       18,
	// One known exception from this order: for SQL Warehouses, CAN_USE and CAN_RUN cannot be ordered and must be upgraded to CAN_MONITOR.
	// We're not doing that currently.
}

func GetLevelScore(a string) int {
	score, ok := PermissionOrder[a]
	if ok {
		return score
	}
	maxPrefixLength := 0
	for levelName, levelScore := range PermissionOrder {
		if strings.HasPrefix(a, levelName) && len(levelName) > maxPrefixLength {
			score = levelScore
			maxPrefixLength = len(levelName)
		}
	}
	return score
}

func CompareLevels(a, b string) int {
	s1 := GetLevelScore(a)
	s2 := GetLevelScore(b)
	if s1 == s2 {
		return strings.Compare(a, b)
	}
	return s1 - s2
}

func GetMaxLevel(a, b string) string {
	if CompareLevels(a, b) >= 0 {
		return a
	}
	return b
}
