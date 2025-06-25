import os

from databricks.sdk import WorkspaceClient
from flask import Flask

app = Flask(__name__)


@app.route("/")
def home():
    job_id = os.getenv("JOB_ID")

    w = WorkspaceClient()
    job = w.jobs.get(job_id)
    return job.settings.name
