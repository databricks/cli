"""Config/wiring coverage for every resource kind without a dedicated typed handle.

These kinds have no cross-resource references to encode, so they're covered uniformly through
the generic env.resource(kind, name) handle — one example per kind, each asserting the
resource exists and a real declared field (never a no-op). The kinds with special wiring
(pipelines, dashboards, genie_spaces, quality_monitors, vector_search_indexes,
model_serving_endpoints, apps) have their own dedicated test files.

CAN test locally: existence + any statically declared field.
CANNOT test locally — needs the cloud backend: that the workspace accepted the config,
server-defaulted/normalized values, and any live state.
"""


# --- MLflow / models ---
def test_model(env):
    assert env.resource("models", "orders_model").config["name"] == "orders_model"


def test_experiment(env):
    assert env.resource("experiments", "orders_experiment").config["name"] == "/Shared/orders_experiment"


def test_registered_model(env):
    rm = env.resource("registered_models", "orders_registered")
    assert rm.config["catalog_name"] == "shop"
    assert rm.config["schema_name"] == "ml"


# --- Unity Catalog ---
def test_schema(env):
    assert env.resource("schemas", "orders_schema").config["catalog_name"] == "shop"


def test_volume(env):
    vol = env.volume("raw_data")
    assert vol.exists()
    assert vol.config["catalog_name"] == "shop"
    assert vol.config["volume_type"] == "MANAGED"


def test_external_location(env):
    assert env.resource("external_locations", "orders_location").config["url"] == "s3://example-bucket/orders"


def test_secret(env):
    assert env.resource("secrets", "orders_secret").config["schema_name"] == "secrets"


# --- Compute ---
def test_cluster(env):
    assert env.resource("clusters", "orders_cluster").config["node_type_id"] == "i3.xlarge"


def test_instance_pool(env):
    assert env.resource("instance_pools", "orders_pool").config["node_type_id"] == "i3.xlarge"


def test_cluster_policy(env):
    assert env.resource("cluster_policies", "orders_policy").config["name"] == "orders-policy"


def test_sql_warehouse(env):
    assert env.resource("sql_warehouses", "orders_warehouse").config["cluster_size"] == "Small"


# --- SQL / secrets / alerts ---
def test_secret_scope(env):
    assert env.resource("secret_scopes", "orders_scope").config["name"] == "orders-scope"


def test_alert(env):
    assert env.resource("alerts", "orders_alert").config["display_name"] == "Orders alert"


def test_vector_search_endpoint(env):
    assert env.resource("vector_search_endpoints", "orders_vs_endpoint").config["endpoint_type"] == "STANDARD"


# --- Databases (Lakebase) ---
def test_database_instance(env):
    assert env.resource("database_instances", "orders_db_instance").config["name"] == "orders-db-instance"


def test_database_catalog(env):
    assert env.resource("database_catalogs", "orders_db_catalog").config["name"] == "orders-db-catalog"


def test_synced_database_table(env):
    assert env.resource("synced_database_tables", "orders_synced").config["name"] == "shop.gold.orders_synced"


# --- Postgres family: same compact shape, keyed on each kind's own id field ---
def test_postgres_project(env):
    assert env.resource("postgres_projects", "orders_pg_project").config["project_id"] == "orders-proj"


def test_postgres_branch(env):
    assert env.resource("postgres_branches", "orders_pg_branch").config["parent"] == "projects/orders-proj"


def test_postgres_endpoint(env):
    assert env.resource("postgres_endpoints", "orders_pg_endpoint").config["endpoint_id"] == "ep1"


def test_postgres_catalog(env):
    assert env.resource("postgres_catalogs", "orders_pg_catalog").config["catalog_id"] == "shop_pg"


def test_postgres_database(env):
    assert env.resource("postgres_databases", "orders_pg_database").config["database_id"] == "orders"


def test_postgres_role(env):
    assert env.resource("postgres_roles", "orders_pg_role").config["role_id"] == "app"


def test_postgres_synced_table(env):
    assert (
        env.resource("postgres_synced_tables", "orders_pg_synced").config["synced_table_id"]
        == "shop.gold.orders_synced_pg"
    )


def test_postgres_snapshot_schedule(env):
    assert (
        env.resource("postgres_snapshot_schedules", "orders_pg_snapshot").config["branch"]
        == "projects/orders-proj/branches/main"
    )
