package iamutil

import (
	"github.com/databricks/cli/libs/sdktypes/scim/v1"
	"github.com/databricks/cli/libs/utils"
	"github.com/google/uuid"
)

// Determines whether a given user is a service principal.
// This function uses a heuristic: if the user name is a UUID, then we assume
// it's a service principal. Unfortunately, the service principal listing API is too
// slow for our purposes. And the "users" and "service principals get" APIs
// only allow access by workspace admins.
func IsServicePrincipal(user *scim.User) bool {
	return IsServicePrincipalName(utils.Deref(user.UserName))
}

func IsServicePrincipalName(userName string) bool {
	_, err := uuid.Parse(userName)
	return err == nil
}
