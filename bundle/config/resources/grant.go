package resources

// Grant holds the grant level settings for a single principal in Unity Catalog.
// Multiple of these can be defined on any resource.
type Grant struct {
	Privileges []string `json:"privileges"`

	Principal string `json:"principal"`
}
