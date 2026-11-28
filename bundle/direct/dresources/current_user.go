package dresources

import "context"

type currentUserType struct{}

// WithCurrentUser attaches the deploying principal's user name (workspace.current_user.user_name:
// a user's email, or a service principal's application id), which the bundle resolves once
// before the deploy, so resources need not ask the API for it again.
func WithCurrentUser(ctx context.Context, userName string) context.Context {
	return context.WithValue(ctx, currentUserType{}, userName)
}

// currentUser returns the user name attached by [WithCurrentUser], or "" if none.
func currentUser(ctx context.Context) string {
	userName, _ := ctx.Value(currentUserType{}).(string)
	return userName
}
