package ucm

import (
	"context"
	"net/http"
)

type UserFingerprint struct {
	Host       string `json:"host"`
	AuthHeader string `json:"auth_header"`
}

func (f *UserFingerprint) IsEmpty() bool {
	return f.Host == "" && f.AuthHeader == ""
}

func (u *Ucm) GetUserFingerprint(ctx context.Context) UserFingerprint {
	return UserFingerprint{
		Host:       u.WorkspaceClient().Config.Host,
		AuthHeader: u.getAuthorizationHeader(),
	}
}

// getAuthorizationHeader extracts the Authorization header from the workspace client configuration.
// If it fails to authenticate, it returns an empty string.
func (u *Ucm) getAuthorizationHeader() string {
	// Create a dummy request to extract the Authorization header
	req := &http.Request{Header: http.Header{}}
	if err := u.WorkspaceClient().Config.Authenticate(req); err != nil {
		return ""
	}

	return req.Header.Get("Authorization")
}
