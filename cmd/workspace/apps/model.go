package apps

type DeployAppRequest struct {
	Manifest  Manifest  `json:"manifest"`
	Resources Resources `json:"resources"`
}

type Manifest map[string]any

type Resources []map[string]any

type DeploymentResponse struct {
	DeploymentId string `json:"deployment_id"`
	State        string `json:"state"`
}

type DeleteAppRequest struct {
	Name string `json:"name"`
}
type DeleteResponse map[string]any

type GetAppRequest struct {
	Name string `json:"name"`
}

type GetResponse map[string]any
