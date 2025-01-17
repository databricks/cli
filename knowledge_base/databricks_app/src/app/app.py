import os

from databricks.sdk import WorkspaceClient
from flask import Flask, render_template, redirect, url_for

app = Flask(__name__)


@app.route("/")
def home():
    job_id = os.getenv("JOB_ID")

    w = WorkspaceClient()
    job = w.jobs.get(job_id)
    runs = w.jobs.list_runs(job_id=job_id)
    return render_template("index.html", job_name=job.settings.name, runs=runs)


@app.route("/run")
def run_job():
    job_id = os.getenv("JOB_ID")

    w = WorkspaceClient()
    w.jobs.run_now(job_id=job_id)
    return redirect(url_for("home"), code=302)
