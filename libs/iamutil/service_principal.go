package iamutil

import (
	"regexp"

	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/google/uuid"
)

var gcpServiceAccountPattern = regexp.MustCompile(`^[^@]+@[^.@\s]+\.iam\.gserviceaccount\.com$`)

// Determines whether a given user is a service principal.
// This function uses heuristics:
// 1. If the user name is a UUID, then we assume it's an Azure service principal
// 2. If the user name matches the GCP service account pattern ([^@]+@[^.@\s]+.iam.gserviceaccount.com),
//    then we assume it's a GCP service account
//
// Unfortunately, the service principal listing API is too slow for our purposes.
// And the "users" and "service principals get" APIs only allow access by workspace admins.
func IsServicePrincipal(user *iam.User) bool {
	// Check if it's an Azure service principal (UUID format)
	_, err := uuid.Parse(user.UserName)
	if err == nil {
		return true
	}
	
	// Check if it's a GCP service account
	return gcpServiceAccountPattern.MatchString(user.UserName)
}
