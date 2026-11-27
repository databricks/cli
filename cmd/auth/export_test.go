package auth

var (
	NewWorkspaceSearcher        = newWorkspaceSearcher
	PromptForWorkspaceSelection = promptForWorkspaceSelection
)

// SetWorkspaceListTimeoutSeconds overrides the workspace lookup budget and returns a restore func.
func SetWorkspaceListTimeoutSeconds(seconds int) func() {
	previous := workspaceListTimeoutSeconds
	workspaceListTimeoutSeconds = seconds
	return func() { workspaceListTimeoutSeconds = previous }
}
