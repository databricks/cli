from dataclasses import replace

from databricks.bundles.secrets import Secret
from databricks.bundles.core import secret_mutator


@secret_mutator
def update_secret(secret: Secret) -> Secret:
    assert isinstance(secret.name, str)

    return replace(secret, name=f"{secret.name} (updated)")
