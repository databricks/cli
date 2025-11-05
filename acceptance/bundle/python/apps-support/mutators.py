from dataclasses import replace

from databricks.bundles.core import app_mutator
from databricks.bundles.apps import App


@app_mutator
def update_app(app: App) -> App:
    assert isinstance(app.name, str)

    return replace(app, name=f"{app.name} (updated)")
