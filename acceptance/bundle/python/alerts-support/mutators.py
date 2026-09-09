from dataclasses import replace

from databricks.bundles.alerts import Alert
from databricks.bundles.core import alert_mutator


@alert_mutator
def update_alert(alert: Alert) -> Alert:
    assert isinstance(alert.display_name, str)

    return replace(alert, display_name=f"{alert.display_name} (updated)")
