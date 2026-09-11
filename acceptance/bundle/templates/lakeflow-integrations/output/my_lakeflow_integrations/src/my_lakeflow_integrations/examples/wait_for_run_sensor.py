import datetime

from databricks.lakeflow.integrations import integration, Context, Sensor, SensorResult
from databricks.sdk import WorkspaceClient


@integration(display_name="Wait for run")
class WaitForRunSensor(Sensor):
    run_id: str

    def __init__(self, run_id: str):
        self.run_id = run_id

    def poll(self, ctx: Context) -> SensorResult:
        from databricks.sdk.runtime import dbutils

        w = WorkspaceClient()

        run = w.jobs.get_run(run_id=int(self.run_id))
        result_state = run.state.result_state if run.state else None

        if result_state is not None:
            dbutils.jobs.taskValues.set("result_state", result_state.name)
            return SensorResult.completed()
        else:
            print("Running", run.run_page_url)
            print("Waiting for a minute...")
            return SensorResult.deferred(duration=datetime.timedelta(minutes=1))
