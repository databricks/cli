package auth

import (
	"github.com/google/uuid"
)

// Determines whether a given user name is a service principal.
// This function uses a heuristic: if the user name is a UUID, then we assume
// it's a service principal. Unfortunately, the service principal listing API is too
// slow for our purposes. And the "users" and "service principals get" APIs
// only allow access by workspace admins.
func IsServicePrincipal(userName string) bool {
	_, err := uuid.Parse(userName)
	return err == nil
}
