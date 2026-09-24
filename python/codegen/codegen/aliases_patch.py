# Backward compatibility aliases: maps old generated type names to new names, per namespace.
# These are emitted into each namespace's __init__.py as simple assignments.
ALIASES: dict[str, dict[str, str]] = {
    "catalogs": {
        "CatalogGrant": "PrivilegeAssignment",
        "CatalogGrantDict": "PrivilegeAssignmentDict",
        "CatalogGrantParam": "PrivilegeAssignmentParam",
    },
    "schemas": {
        "SchemaGrant": "PrivilegeAssignment",
        "SchemaGrantDict": "PrivilegeAssignmentDict",
        "SchemaGrantParam": "PrivilegeAssignmentParam",
    },
    "volumes": {
        "VolumeGrant": "PrivilegeAssignment",
        "VolumeGrantDict": "PrivilegeAssignmentDict",
        "VolumeGrantParam": "PrivilegeAssignmentParam",
    },
}
