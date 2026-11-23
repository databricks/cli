package dms

import (
	"context"
	"fmt"
	"net/http"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	databricksclient "github.com/databricks/databricks-sdk-go/client"
)

// FeatureRecordHistoryAuto is the enrollment under which deploys record deployment history
// without the bundle asking for it.
const FeatureRecordHistoryAuto = "record_history_auto"

const checkEnrollmentsPath = "/api/2.0/bundle/enrollments:check"

// checkEnrollmentsRequest travels in the query string: the endpoint is a GET, and the SDK
// builds a GET's query string from the `url` tags, repeating the parameter for a slice.
type checkEnrollmentsRequest struct {
	Features []string `url:"features"`
}

type checkEnrollmentsResponse struct {
	Enrollments map[string]bool `json:"enrollments"`
}

// CheckEnrollments reports which of the given rollout features the workspace is enrolled in.
// The service leaves out a feature it does not know, so a missing key reads as not enrolled.
// Asking about no features is answered here: the service rejects an empty list.
//
// TODO: switch to the SDK's generated method once the SDK has it.
func CheckEnrollments(ctx context.Context, w *databricks.WorkspaceClient, features ...string) (map[string]bool, error) {
	if len(features) == 0 {
		return map[string]bool{}, nil
	}
	client, err := databricksclient.New(w.Config)
	if err != nil {
		return nil, err
	}
	var resp checkEnrollmentsResponse
	req := checkEnrollmentsRequest{Features: features}
	err = client.Do(ctx, http.MethodGet, checkEnrollmentsPath, auth.WorkspaceIDHeaders(client.Config), nil, req, &resp)
	if err != nil {
		return nil, fmt.Errorf("checking enrollments: %w", err)
	}
	return resp.Enrollments, nil
}
