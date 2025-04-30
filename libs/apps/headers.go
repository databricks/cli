package apps

import (
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/google/uuid"
)

func GetXHeaders(user *iam.User) map[string]string {
	return map[string]string{
		"X-Forwarded-Host":               "localhost",
		"X-Forwarded-Preferred-Username": user.DisplayName,
		"X-Forwarded-User":               user.UserName,
		"X-Forwarded-Email":              getEmail(user),
		"X-Real-Ip":                      "127.0.0.1",
		"X-Request-Id":                   uuid.New().String(),
	}
}

func getEmail(user *iam.User) string {
	if len(user.Emails) > 0 {
		return user.Emails[0].Value
	}
	return user.UserName
}
