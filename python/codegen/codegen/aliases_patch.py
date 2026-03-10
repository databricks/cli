# Backward compatibility aliases: maps old generated type names to new names, per namespace.
# These are emitted into each namespace's __init__.py as simple assignments.
ALIASES: dict[str, dict[str, str]] = {
    "catalogs": {
        "CatalogGrant": "PrivilegeAssignment",
        "CatalogGrantDict": "PrivilegeAssignmentDict",
        "CatalogGrantParam": "PrivilegeAssignmentParam",
        "CatalogGrantPrivilege": "Privilege",
        "CatalogGrantPrivilegeParam": "PrivilegeParam",
    },
    "schemas": {
        "SchemaGrant": "PrivilegeAssignment",
        "SchemaGrantDict": "PrivilegeAssignmentDict",
        "SchemaGrantParam": "PrivilegeAssignmentParam",
        "SchemaGrantPrivilege": "Privilege",
        "SchemaGrantPrivilegeParam": "PrivilegeParam",
    },
    "volumes": {
        "VolumeGrant": "PrivilegeAssignment",
        "VolumeGrantDict": "PrivilegeAssignmentDict",
        "VolumeGrantParam": "PrivilegeAssignmentParam",
        "VolumeGrantPrivilege": "Privilege",
        "VolumeGrantPrivilegeParam": "PrivilegeParam",
    },
    "jobs": {
        "Permission": "JobPermission",
        "PermissionDict": "JobPermissionDict",
        "PermissionParam": "JobPermissionParam",
    },
    "pipelines": {
        "Permission": "PipelinePermission",
        "PermissionDict": "PipelinePermissionDict",
        "PermissionParam": "PipelinePermissionParam",
    },
}
