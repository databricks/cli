package apiclient

// workspaceIDNone mirrors profile.WorkspaceIDNone; libs/databrickscfg/profile
// cannot be imported here because it (transitively) depends on this package.
const workspaceIDNone = "none"

// AddWorkspaceIDHeader sets the workspace routing header that unified
// ("SPOG") hosts need. The CLI's "none" sentinel means "no workspace ID" and
// is not forwarded.
func (c *Client) AddWorkspaceIDHeader(headers map[string]string) {
	if id := c.Config.WorkspaceID; id != "" && id != workspaceIDNone {
		headers["X-Databricks-Workspace-Id"] = id
	}
}
