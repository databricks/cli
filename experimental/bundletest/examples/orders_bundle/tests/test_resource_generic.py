"""The generic env.resource(kind, name) handle — reaches ANY resource kind.

Every resource kind is reachable through env.resource(kind, name), which reads its declared
config from the bundle. The typed handles (env.pipeline, env.dashboard, ...) add resource-
specific accessors on top of this same base for the kinds that reference tables/artifacts.

CAN test locally:
- any declared resource's existence and its config fields
- KeyError-safe exists() for a resource that isn't declared

CANNOT test locally — needs the cloud backend:
- that the workspace accepted / deployed the config, and server-defaulted values
"""


def test_generic_handle_reads_any_kind(env):
    # A kind with no dedicated typed handle is still fully reachable.
    catalog = env.resource("catalogs", "orders_catalog")
    assert catalog.exists()
    assert catalog.config["name"] == "shop"
    assert catalog.grants() == [{"principal": "users", "privileges": ["USE_CATALOG"]}]


def test_missing_resource_does_not_exist(env):
    # get_resource raises KeyError for an undeclared resource; exists() must not propagate it.
    assert not env.resource("catalogs", "nope").exists()
    assert not env.pipeline("nope").exists()
