package auth

import (
	"github.com/google/uuid"
)

// Determines whether a given user id is a service principal.
// This function uses a heuristic: if the user id is a UUID, then we assume
// it's a service principal. Unfortunately, the service principal listing API is too
// slow for our purposes. And the "users" and "service principals get" APIs
// only allow access by workspace admins.
func IsServicePrincipal(userId string) bool {
	_, err := uuid.Parse(userId)
	return err == nil
}
