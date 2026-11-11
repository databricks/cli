from dataclasses import replace

from databricks.bundles.core import dashboard_mutator
from databricks.bundles.dashboards import Dashboard


@dashboard_mutator
def update_dashboard(dashboard: Dashboard) -> Dashboard:
    assert isinstance(dashboard.display_name, str)

    return replace(dashboard, display_name=f"{dashboard.display_name} (updated)")
