"""score_model — a notebook job (notebook_task).

Non-SQL tasks (notebook / Python / Scala / R) need a real cluster, so the local backend
cannot run them.

CAN test locally:
- nothing runs; the framework refuses and SKIPS with a reason
  (never a silent pass, never a false failure)

CANNOT test locally — needs the cloud backend:
- running the job at all
- any assertion on its output tables

On the cloud backend the same env.run_job(...) + env.table(...) assertions work unchanged,
because assertions inspect the *output*, which is language-agnostic.
"""


def test_notebook_job_skips_locally(env):
    # Reported as SKIPPED with a reason on the local backend.
    env.run_job("score_model")
