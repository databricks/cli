from dataclasses import replace

from databricks.bundles.secret_scopes import SecretScope
from databricks.bundles.core import secret_scope_mutator


@secret_scope_mutator
def update_secret_scope(scope: SecretScope) -> SecretScope:
    assert isinstance(scope.name, str)

    return replace(scope, name=f"{scope.name} (updated)")
