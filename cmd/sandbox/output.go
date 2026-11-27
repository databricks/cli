package sandbox

import (
	"strings"

	"github.com/databricks/databricks-sdk-go/common/types/duration"
	sdk "github.com/databricks/databricks-sdk-go/service/sandbox"
)

// createJSONOutput is the established JSON shape for sandbox create.
type createJSONOutput struct {
	SandboxID string `json:"sandboxId"`
	Status    string `json:"status"`
}

// sandboxJSONOutput is the established list/status JSON shape.
// IdleTimeout and NoAutostop are pointer-typed so we can distinguish
// "field absent on the wire" (server uses its default) from "explicitly
// set to 0 / false". The SDK duration preserves its protobuf JSON encoding.
type sandboxJSONOutput struct {
	SandboxID   string             `json:"sandboxId"`
	Status      string             `json:"status"`
	Name        string             `json:"name,omitempty"`
	IdleTimeout *duration.Duration `json:"idleTimeout,omitempty"`
	NoAutostop  *bool              `json:"noAutostop,omitempty"`
}

func sandboxResourceID(sb *sdk.Sandbox) string {
	return strings.TrimPrefix(sb.Name, "sandboxes/")
}

func sandboxLabel(sb *sdk.Sandbox) string {
	if label := strings.TrimSpace(sb.DisplayName); label != "" {
		return label
	}
	return sandboxResourceID(sb)
}

func sandboxStatus(sb *sdk.Sandbox) string {
	if sb.Status != nil {
		switch sb.Status.State {
		case sdk.SandboxStateSandboxStatePending:
			return "Creating"
		case sdk.SandboxStateSandboxStateRunning:
			return "Running"
		case sdk.SandboxStateSandboxStateStopping:
			return "Stopping"
		case sdk.SandboxStateSandboxStateStopped:
			return "Stopped"
		}
	}
	return "Unknown"
}

func autoStopLabel(sb *sdk.Sandbox) string {
	if sb.Spec != nil && sb.Spec.Compute != nil && sb.Spec.Compute.InactivityTimeout != nil {
		if secs := int64(sb.Spec.Compute.InactivityTimeout.AsDuration().Seconds()); secs > 0 {
			return formatDurationSecs(secs)
		}
	}
	return "never"
}

// Keep the published JSON field names for scripts; lifecycle code uses SDK values directly.
func sandboxJSON(sb *sdk.Sandbox) *sandboxJSONOutput {
	entry := &sandboxJSONOutput{SandboxID: sandboxResourceID(sb), Status: sandboxStatus(sb), Name: sandboxLabel(sb)}
	if sb.Spec != nil && sb.Spec.Compute != nil && sb.Spec.Compute.InactivityTimeout != nil {
		timeout := sb.Spec.Compute.InactivityTimeout
		entry.IdleTimeout = timeout
		disabled := timeout.AsDuration() == 0
		entry.NoAutostop = &disabled
	}
	return entry
}
