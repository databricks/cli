from dataclasses import replace

from databricks.bundles.data_quality_monitors import DataQualityMonitor
from databricks.bundles.core import data_quality_monitor_mutator


@data_quality_monitor_mutator
def update_data_quality_monitor(
    data_quality_monitor: DataQualityMonitor,
) -> DataQualityMonitor:
    assert isinstance(data_quality_monitor.object_id, str)

    return replace(data_quality_monitor, object_id=f"{data_quality_monitor.object_id} (updated)")
