from dataclasses import replace

from databricks.bundles.core import quality_monitor_mutator
from databricks.bundles.quality_monitors import QualityMonitor


@quality_monitor_mutator
def update_quality_monitor(monitor: QualityMonitor) -> QualityMonitor:
    assert isinstance(monitor.output_schema_name, str)

    return replace(
        monitor, output_schema_name=f"{monitor.output_schema_name} (updated)"
    )
