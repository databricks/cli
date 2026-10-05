package testserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type aiTrainingValidateConfigRequest struct {
	Task       *aiTrainingTask       `json:"task"`
	RunOptions *aiTrainingRunOptions `json:"run_options,omitempty"`
}

type aiTrainingTask struct {
	Experiment                string                 `json:"experiment"`
	Deployments               []aiTrainingDeployment `json:"deployments"`
	PriorityClass             *string                `json:"priority_class,omitempty"`
	MLflowRun                 *string                `json:"mlflow_run,omitempty"`
	MLflowExperimentDirectory *string                `json:"mlflow_experiment_directory,omitempty"`
	MLflowArtifactLocation    *string                `json:"mlflow_artifact_location,omitempty"`
}

type aiTrainingDeployment struct {
	CommandPath string             `json:"command_path"`
	Compute     *aiTrainingCompute `json:"compute"`
}

type aiTrainingCompute struct {
	AcceleratorType       string  `json:"accelerator_type"`
	AcceleratorCount      *int    `json:"accelerator_count"`
	ProvisionedCapacityID *string `json:"provisioned_capacity_id,omitempty"`
}

type aiTrainingRunOptions struct {
	MaxRetries       *int              `json:"max_retries,omitempty"`
	TimeoutMinutes   *int              `json:"timeout_minutes,omitempty"`
	IdempotencyToken *string           `json:"idempotency_token,omitempty"`
	UsagePolicyName  *string           `json:"usage_policy_name,omitempty"`
	UsagePolicyID    *string           `json:"usage_policy_id,omitempty"`
	EnvVariables     map[string]string `json:"env_variables,omitempty"`
	Secrets          map[string]string `json:"secrets,omitempty"`
}

type aiTrainingFieldError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type aiTrainingValidateConfigResponse struct {
	Errors []aiTrainingFieldError `json:"errors,omitempty"`
}

// aiTrainingValidateConfig is a request-contract stub, not an implementation of
// production validation rules. Acceptance fixtures provide scenario-specific
// responses when they need validation failures.
func aiTrainingValidateConfig(req Request) any {
	var request aiTrainingValidateConfigRequest
	decoder := json.NewDecoder(bytes.NewReader(req.Body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return invalidAiTrainingValidateConfigRequest(err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return invalidAiTrainingValidateConfigRequest(errors.New("request must contain exactly one JSON value"))
	}

	return aiTrainingValidateConfigResponse{}
}

func invalidAiTrainingValidateConfigRequest(err error) Response {
	return Response{
		StatusCode: http.StatusBadRequest,
		Body: map[string]string{
			"error_code": "INVALID_PARAMETER_VALUE",
			"message":    fmt.Sprintf("invalid config validation request: %s", err),
		},
	}
}
