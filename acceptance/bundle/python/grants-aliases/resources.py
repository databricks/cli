from databricks.bundles.core import Resources

# New unified types (preferred going forward)
from databricks.bundles.schemas import Privilege, PrivilegeAssignment

# Old per-resource aliases (kept for backward compatibility)
from databricks.bundles.schemas import SchemaGrant, SchemaGrantPrivilege


def load_resources() -> Resources:
    resources = Resources()

    resources.add_schema(
        "my_schema",
        {
            "name": "my_schema",
            "catalog_name": "my_catalog",
            # New type
            "grants": [
                PrivilegeAssignment(
                    principal="data-team@example.com",
                    privileges=[Privilege.USE_SCHEMA],
                )
            ],
        },
    )

    resources.add_schema(
        "my_schema_legacy",
        {
            "name": "my_schema_legacy",
            "catalog_name": "my_catalog",
            # Old alias type — identical to PrivilegeAssignment at runtime
            "grants": [
                SchemaGrant(
                    principal="data-team@example.com",
                    privileges=[SchemaGrantPrivilege.USE_SCHEMA],
                )
            ],
        },
    )

    return resources
