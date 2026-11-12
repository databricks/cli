Acceptance tests are blackbox tests that are run against compiled binary.

## Where tests run

**Every test runs locally**, against a "fake" HTTP server pretending to be Databricks API (`libs/testserver`). This is what `./task test` does and it covers the whole `acceptance` tree.

A test can *additionally* opt into running against a real workspace by setting `Cloud = true` in its `test.toml`. That is an extra run, not a different one: `Cloud = true` never means "this test does not run locally". Related settings:

- `CloudSlow = true` only narrows a `Cloud = true` test: its cloud run is skipped when `-short` is passed. On its own (without `Cloud = true`) it does not enable the cloud run.
- `CloudEnvs` only narrows the cloud run further. It is ignored locally.
- `Cloud` is inherited from parent `test.toml` files, so a whole subtree can be opted in at once. The root `acceptance/test.toml` defaults it to `Cloud = false`, i.e. local only.
- Each test's effective settings are visible in its generated `out.test.toml`.

The cloud run is keyed off `CLOUD_ENV`; see `getSkipReason` in `acceptance/acceptance_test.go` for the exact gating. What skips a test *locally* is a different set of settings entirely: `GOOS`, `RunsOnDbr`, and `DATABRICKS_TEST_SELECT_CHANGED` (used by cloud PR runs to cover only the tests the branch touches).

To run tests against a real workspace: `deco env run -i -n aws-prod-ucws -- <go test command>` (requires the `deco` tool and access to a test env).

### Running named tests on a PR

PR runs select only the tests the branch changed. To run more, name them in the **last** commit message of the PR, each line starting with a cloud prefix and listing one or more whitespace-separated test names:

```
empty commit to run tests

aws: TestAccept/bundle/resources/jobs
cloud: TestAccept/bundle/resources/pipelines TestAccept/cmd/fs/cp/file-to-file
gcp: TestAccept/bundle/validate/enum/DMS=true
```

- `aws:`, `azure:`, `gcp:` run the test on that cloud only; `cloud:` runs it on every cloud. Prefixes are lowercase and must start the line (surrounding whitespace is ignored). Lines with any other prefix are ignored. A line with a word that does not start with `Test` is not read either (so `aws: fix flaky TestAccept retries` is prose, not a request to run the suite), and is logged as a warning, as is ``aws: `TestAccept/x` ``.
- Every named test is also selected for the local (testserver) runs, whatever its prefix.
- `TestAccept/<path>` selects every test dir that is `<path>` or under `<path>/`, with all its variants. `TestAccept/<dir>/KEY=value[/KEY=value...]` selects only the matching variants of test dir `<dir>`, as `go test` names them; each `KEY=value` must be in one of the dir's variants (per its `out.test.toml`). `TestAccept` alone selects every test. A name that matches no acceptance test (including non-acceptance test names) is logged as a warning and otherwise ignored.
- The named tests are added to the changed-test selection (`DATABRICKS_TEST_SELECT_CHANGED`, used by the PR cloud run and `./task local-select`) and to the windows/macOS PR subset (`TESTS_SELECT_SUBSET_PCT`), without counting toward their limits. Runs with neither selection (Linux PR cell, nightly) do not read the commit message: they run every test.
- On cloud, a named test still needs `Cloud = true` (and a matching `CloudEnvs`), but it runs even if it is `CloudSlow` when it is named by its own dir or variant name (naming any variant of a dir lifts this for the dir). A path above the dir (or a bare `TestAccept`) does not lift `CloudSlow`, so a whole subtree stays within the PR run's `-short` budget.
- On pull requests, such a line makes `tools/testmask` select the tests and the cloud run even if the changed files alone would not. The merge queue ignores it.
- On the merge commit GitHub creates for a PR (`refs/pull/N/merge`), the message of the PR's head commit is used. The parser is in `acceptance/internal/selection/commit.go`.
- To see what a commit message selects without running the suite: `go run ./acceptance/internal/selection/cmd [-cloud aws|azure|gcp]` prints the named tests and the warnings after the changed ones.

## Authoring

To author a test,
 - Add a new directory under `acceptance`. Any level of nesting is supported.
 - Add `databricks.yml` there.
 - Add `script` with commands to run, e.g. `$CLI bundle validate`. The test case is recognized by presence of `script`.

The test runner will run script and capture output and compare it with `output.txt` file in the same directory.

In order to write `output.txt` for the first time or overwrite it with the current output pass -update flag to go test.

The scripts are run with `bash -e` so any errors will be propagated. They are captured in `output.txt` by appending `Exit code: N` line at the end.

For more complex tests one can also use:
- `errcode` helper: if the command fails with non-zero code, it appends `Exit code: N` to the output but returns success to caller (bash), allowing continuation of script.
- `musterr` helper: runs the command and fails the test if it *succeeds*; on the expected failure the script continues. Use it to assert a command must error.
- `trace` helper: prints the arguments before executing the command.
- `title` helper: prints a `=== <text>` section header to label a phase of the script.
- custom output files: redirect output to custom file (it must start with `out`), e.g. `$CLI bundle validate > out.txt 2> out.error.txt`.

The complete set of shell helpers (the above plus `withdir`, `git-repo-init`, `uuid`, `sethome`, and others) is defined in `acceptance/script.prepare`.

Any file starting with "LOG" will be logged to test log (visible with go test -v).

See [selftest](./selftest) for more examples.

## Benchmarks

Benchmarks are regular acceptance test that log measurements in certain format. The output can be fed to `tools/bench_parse.py` to print a summary table.

Test runner recognizes benchmark as having "benchmark" anywhere in the path. For these tests parallel execution is disabled if and only if BENCHMARK\_PARAMS variable is set.

The benchmarks make use of two scripts:

- `gen_config.py —jobs N` to generate a config with N jobs
- `benchmark.py` command to run command a few times and log the time measurements.

The default number of runs in benchmark.py depends on BENCHMARK\_PARAMS variable. If it’s set, the default number is 5. Otherwise it is 1.

## Running acceptance tests on Windows

To run the acceptance tests from a terminal on Windows (eg. Git Bash from VS Code),
you need to install a few prerequisites and optionally make user policy changes.

These steps were verified to work with a Windows 11 VM running on Parallels.

### Install Chocolatey

Run "PowerShell" as administrator and follow the [Chocolatey installation instructions][choco].

[choco]: https://chocolatey.org/install#individual

Confirm it is installed correctly:
```pwsh
PS C:\WINDOWS\system32> choco --version
2.5.1
```

### Tools

Install the following tools:
```pwsh
choco install -y vscode
choco install -y git
choco install -y make
choco install -y jq
choco install -y python3
choco install -y uv
choco install -y go
choco install -y nodejs
```

### Shim for `python3.exe`

The default Python installation only installs `python.exe` and not `python3.exe`.

We rely on calling `python3` in acceptance tests (shebangs in scripts and elsewhere).

To install `python3` and `pip3` shims for the install, run PowerShell as administrator and execute the following:
```pwsh
# Refresh first to pick up Python 3 installed in the previous step.
refreshenv

# Optional: python3, only if python.exe exists
$python3Exists = $false
try {
  $py = (Get-Command python.exe -ErrorAction Stop).Source
  $python3Exists = $true
  & "$env:ChocolateyInstall\tools\shimgen.exe" `
    --output "$env:ChocolateyInstall\bin\python3.exe" `
    --path   $py
} catch {
  Write-Host "python.exe not found, skipping python3 shim creation."
}

# Optional: pip3, too, but only if pip.exe exists
$pipExists = $false
try {
  $pip = (Get-Command pip.exe -ErrorAction Stop).Source
  $pipExists = $true
  & "$env:ChocolateyInstall\tools\shimgen.exe" `
    --output "$env:ChocolateyInstall\bin\pip3.exe" `
    --path   $pip
} catch {
  Write-Host "pip.exe not found, skipping pip3 shim creation."
}

# Refresh to pick up the shims.
refreshenv

# Check python3 version only if python3 shim was created
if ($python3Exists) {
  try {
    python3 --version
  } catch {
    Write-Host "python3 not found or not working. Please check your installation."
  }
} else {
  Write-Host "python3 not available."
}

# Check pip3 version only if pip3 shim was created
if ($pipExists) {
  try {
    pip3 --version
  } catch {
    Write-Host "pip3 not found or not working. Please check your installation."
  }
} else {
  Write-Host "pip3 not available."
}
```

### Enable symlink creation

You need to be able to create symlinks.
If you're not an administrator user, enable this by following these steps:

* Press Win+R, type `secpol.msc`, press Enter.
* Go to Local Policies → User Rights Assignment.
* Find "Create symbolic links".
* Add your username to the list.
* Reboot.

### Enable long path support (up to ~32,767 characters)

Some acceptance tests fail if this is not enabled because their paths
exceed the default maximum total length of 260 characters.

* Press Win+R, type `gpedit.msc`, press Enter.
* Go to Computer Configuration → Administrative Templates → System → Filesystem → Enable Win32 long paths.
* Enable the setting.
* Reboot.

### Disable Microsoft Defender

The tests frequently create and remove temporary directories.
Sometimes, Microsoft Defender locks a file (such as an executable) during deletion,
which can cause errors and test failures.

* Press Win+R, type `gpedit.msc`, press Enter.
* Go to Computer Configuration → Administrative Templates → Windows Components → Microsoft Defender Antivirus → Turn off Microsoft Defender Antivirus.
* Enable the setting.
* Reboot.
