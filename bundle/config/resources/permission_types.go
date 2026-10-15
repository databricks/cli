package resources

// Each resource defines its own permission type so that the JSON schema names them distinctly.
// Using non-alias type definitions (not =) makes them appear as named types in the schema.
// The underlying struct is identical to PermissionT[L], enabling conversion to use generic methods.

// Permission is used for resources that use the generic PermissionLevel (Alert, Dashboard, DatabaseInstance, PostgresProject).
type Permission PermissionT[PermissionLevel]

func (p Permission) String() string {
	return PermissionT[PermissionLevel](p).String()
}

// If the SDK exposes a resource's permission level, add it here.
type (
	AppPermission                  PermissionT[AppPermissionLevel]
	ClusterPermission              PermissionT[ClusterPermissionLevel]
	ClusterPolicyPermission        PermissionT[ClusterPolicyPermissionLevel]
	InstancePoolPermission         PermissionT[InstancePoolPermissionLevel]
	JobPermission                  PermissionT[JobPermissionLevel]
	MlflowExperimentPermission     PermissionT[ExperimentPermissionLevel]
	MlflowModelPermission          PermissionT[RegisteredModelPermissionLevel]
	ModelServingEndpointPermission PermissionT[ServingEndpointPermissionLevel]
	PipelinePermission             PermissionT[PipelinePermissionLevel]
	SqlWarehousePermission         PermissionT[WarehousePermissionLevel]
	VectorSearchEndpointPermission PermissionT[VectorSearchEndpointPermissionLevel]
)
