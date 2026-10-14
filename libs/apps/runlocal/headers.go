package runlocal

import (
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/scim/v1"
	"github.com/google/uuid"
)

func GetXHeaders(user *scim.User) map[string]string {
	return map[string]string{
		"X-Forwarded-Host":               "localhost",
		"X-Forwarded-Preferred-Username": utils.Deref(user.DisplayName),
		"X-Forwarded-User":               utils.Deref(user.UserName),
		"X-Forwarded-Email":              getEmail(user),
		"X-Real-Ip":                      "127.0.0.1",
		"X-Request-Id":                   uuid.New().String(),
	}
}

func getEmail(user *scim.User) string {
	if len(user.Emails) > 0 {
		return utils.Deref(user.Emails[0].Value)
	}
	return utils.Deref(user.UserName)
}
