direct: the experimental `job_runs` resource now waits for the run to finish, exposes its output fields (e.g. `${resources.job_runs.<name>.state.result_state}`), and dedupes runs on deploy retries.
