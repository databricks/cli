package dsc

import (
	"encoding/json"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"reflect"
)

func init() {
	RegisterResourceWithMetadata("Databricks.DSC/User", &UserHandler{}, userMetadata())
}

// ============================================================================
// Property Descriptions (from SDK documentation)
// ============================================================================

var userPropertyDescriptions = PropertyDescriptions{
	"id":           "Databricks user ID. This is the unique identifier for the user.",
	"user_name":    "Email address of the Databricks user. This is used as the primary identifier.",
	"display_name": "String that represents a concatenation of given and family names.",
	"active":       "If this user is active.",
	"external_id":  "External ID is reserved for future use.",
	"emails":       "All the emails associated with the Databricks user.",
	"entitlements": "Entitlements assigned to the user.",
	"groups":       "Groups the user belongs to.",
	"roles":        "Corresponds to AWS instance profile/arn role.",
	"name":         "The name of the user (given name and family name).",
}

// ============================================================================
// Metadata definition
// ============================================================================

func userMetadata() ResourceMetadata {
	return BuildMetadata(MetadataConfig{
		ResourceType:      "Databricks.DSC/User",
		Description:       "Manage Databricks workspace users",
		SchemaDescription: "Schema for managing Databricks workspace users.",
		ResourceName:      "user",
		Tags:              []string{"databricks", "user", "iam", "workspace"},
		Descriptions:      userPropertyDescriptions,
		SchemaType:        reflect.TypeOf(iam.User{}),
	})
}

// ============================================================================
// User resource handler
// ============================================================================

type UserState struct {
	ID          string `json:"id"`
	UserName    string `json:"user_name"`
	DisplayName string `json:"display_name,omitempty"`
	Active      bool   `json:"active"`
}

type UserHandler struct{}

func (h *UserHandler) Get(ctx ResourceContext, input json.RawMessage) (any, error) {
	req, err := unmarshalInput[iam.User](input)
	if err != nil {
		return nil, err
	}

	cmdCtx, w := getWorkspaceClient(ctx)

	if req.Id != "" {
		user, err := w.Users.GetById(cmdCtx, req.Id)
		if err != nil {
			return nil, err
		}
		return userToState(user), nil
	}

	if req.UserName != "" {
		if err := validateRequired(RequiredField{"user_name", req.UserName}); err != nil {
			return nil, err
		}

		users := w.Users.List(cmdCtx, iam.ListUsersRequest{
			Filter: "userName eq \"" + req.UserName + "\"",
		})

		user, err := users.Next(cmdCtx)
		if err != nil {
			return nil, NotFoundError("user", "user_name="+req.UserName)
		}
		return userToState(&user), nil
	}

	return nil, NotFoundError("user", "id or user_name must be provided")
}

func (h *UserHandler) Set(ctx ResourceContext, input json.RawMessage) error {
	req, err := unmarshalInput[iam.User](input)
	if err != nil {
		return err
	}

	cmdCtx, w := getWorkspaceClient(ctx)

	if req.Id != "" {
		return w.Users.Update(cmdCtx, req)
	}

	if req.UserName != "" {
		users := w.Users.List(cmdCtx, iam.ListUsersRequest{
			Filter: "userName eq \"" + req.UserName + "\"",
		})

		existingUser, err := users.Next(cmdCtx)
		if err == nil && existingUser.Id != "" {
			req.Id = existingUser.Id
			return w.Users.Update(cmdCtx, req)
		}

		_, err = w.Users.Create(cmdCtx, req)
		return err
	}

	return validateRequired(RequiredField{"user_name", ""})
}

func (h *UserHandler) Delete(ctx ResourceContext, input json.RawMessage) error {
	req, err := unmarshalInput[iam.User](input)
	if err != nil {
		return err
	}

	cmdCtx, w := getWorkspaceClient(ctx)

	if req.Id != "" {
		return w.Users.DeleteById(cmdCtx, req.Id)
	}

	if req.UserName != "" {
		users := w.Users.List(cmdCtx, iam.ListUsersRequest{
			Filter: "userName eq \"" + req.UserName + "\"",
		})

		user, err := users.Next(cmdCtx)
		if err != nil {
			return NotFoundError("user", "user_name="+req.UserName)
		}
		return w.Users.DeleteById(cmdCtx, user.Id)
	}

	return validateRequired(RequiredField{"id or user_name", ""})
}

func (h *UserHandler) Export(ctx ResourceContext) (any, error) {
	cmdCtx, w := getWorkspaceClient(ctx)

	var allUsers []UserState

	users := w.Users.List(cmdCtx, iam.ListUsersRequest{})
	for {
		user, err := users.Next(cmdCtx)
		if err != nil {
			break
		}
		allUsers = append(allUsers, userToState(&user))
	}

	return allUsers, nil
}

func userToState(user *iam.User) UserState {
	return UserState{
		ID:          user.Id,
		UserName:    user.UserName,
		DisplayName: user.DisplayName,
		Active:      user.Active,
	}
}
