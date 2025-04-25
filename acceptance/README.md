Acceptance tests are blackbox tests that are run against compiled binary.

Currently these tests are run against "fake" HTTP server pretending to be Databricks API. However, they will be extended to run against real environment as regular integration tests.

To author a test,
 - Add a new directory under `acceptance`. Any level of nesting is supported.
 - Add `databricks.yml` there.
 - Add `script` with commands to run, e.g. `$CLI bundle validate`. The test case is recognized by presence of `script`.

The test runner will run script and capture output and compare it with `output.txt` file in the same directory.

In order to write `output.txt` for the first time or overwrite it with the current output pass -update flag to go test.

The scripts are run with `bash -e` so any errors will be propagated. They are captured in `output.txt` by appending `Exit code: N` line at the end.

For more complex tests one can also use:
- `errcode` helper: if the command fails with non-zero code, it appends `Exit code: N` to the output but returns success to caller (bash), allowing continuation of script.
- `trace` helper: prints the arguments before executing the command.
- custom output files: redirect output to custom file (it must start with `out`), e.g. `$CLI bundle validate > out.txt 2> out.error.txt`.

See [selftest](./selftest) for a toy test.
