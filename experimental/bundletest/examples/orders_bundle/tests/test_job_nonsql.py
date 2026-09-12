"""score_model — a notebook job (notebook_task).

Non-SQL tasks (notebook / Python / Scala / R) need a real cluster, so the local backend
cannot run them.

CAN test locally:
- that the framework refuses to run it and signals LocalUnsupported
  (in a normal run this surfaces as a SKIP with a reason — never a silent pass or false fail)

CANNOT test locally — needs the cloud backend:
- running the job at all
- any assertion on its output tables

On the cloud backend the same env.run_job(...) + env.table(...) assertions work unchanged,
because assertions inspect the *output*, which is language-agnostic.
"""

import pytest
from bundletest import LocalUnsupported


def test_notebook_job_is_refused_locally(env):
    # Normal user code is just `env.run_job("score_model")`, which auto-skips. Here we
    # assert the guard fires so the boundary itself is covered.
    with pytest.raises(LocalUnsupported):
        env.run_job("score_model")
