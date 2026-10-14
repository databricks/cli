package iamutil

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/scim/v1"
	"github.com/databricks/sdk-go/core/apierr"
	sdkscim "github.com/databricks/sdk-go/scim/v1"
)

const meRetryTimeout = 30 * time.Second

// meRetryInterval is the pause between attempts while the SCIM Me endpoint
// returns 500, which it does transiently right after workspace creation.
const meRetryInterval = time.Second

// GetCurrentUser returns the authenticated user. It is returned as the
// JSON-capable mirror type because callers store it in bundle configuration.
func GetCurrentUser(ctx context.Context, w *sdkclients.Clients) (*scim.User, error) {
	deadline := time.Now().Add(meRetryTimeout)
	for {
		user, err := w.Scim.Me(ctx, sdkscim.MeRequest{})
		if err == nil {
			return UserFromSDK(user)
		}
		apiErr, ok := errors.AsType[*apierr.APIError](err)
		if !ok || apiErr.HTTPStatusCode() != http.StatusInternalServerError || time.Now().Add(meRetryInterval).After(deadline) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(meRetryInterval):
		}
	}
}

// UserFromSDK converts an sdk-go user into its JSON-capable mirror.
func UserFromSDK(u *sdkscim.User) (*scim.User, error) {
	if u == nil {
		return nil, nil
	}
	convert := func(field string, values []sdkscim.ComplexValue) ([]scim.ComplexValue, error) {
		if values == nil {
			return nil, nil
		}
		out := make([]scim.ComplexValue, len(values))
		for i := range values {
			v, err := scim.ComplexValueFromSDK(&values[i])
			if err != nil {
				return nil, fmt.Errorf("User.%s: %w", field, err)
			}
			out[i] = *v
		}
		return out, nil
	}
	emails, err := convert("Emails", u.Emails)
	if err != nil {
		return nil, err
	}
	entitlements, err := convert("Entitlements", u.Entitlements)
	if err != nil {
		return nil, err
	}
	groups, err := convert("Groups", u.Groups)
	if err != nil {
		return nil, err
	}
	roles, err := convert("Roles", u.Roles)
	if err != nil {
		return nil, err
	}
	name, err := scim.NameFromSDK(u.Name)
	if err != nil {
		return nil, fmt.Errorf("User.Name: %w", err)
	}
	return &scim.User{
		Active:       u.Active,
		DisplayName:  u.DisplayName,
		Emails:       emails,
		Entitlements: entitlements,
		ExternalId:   u.ExternalId,
		Groups:       groups,
		Id:           u.Id,
		Name:         name,
		Roles:        roles,
		Schemas:      u.Schemas,
		UserName:     u.UserName,
	}, nil
}
