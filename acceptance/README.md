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
- `musterr` helper: runs the command and fails the test if it *succeeds*; on the expected failure the script continues. Use it to assert a command must error.
- `trace` helper: prints the arguments before executing the command.
- `title` helper: prints a `=== <text>` section header to label a phase of the script.
- custom output files: redirect output to custom file (it must start with `out`), e.g. `$CLI bundle validate > out.txt 2> out.error.txt`.

The complete set of shell helpers (the above plus `withdir`, `git-repo-init`, `uuid`, `sethome`, and others) is defined in `acceptance/script.prepare`.

Any file starting with "LOG" will be logged to test log (visible with go test -v).

See [selftest](./selftest) for more examples.

## Soft-failing volatile goldens

Some goldens drift for reasons outside our control: a live backend rewords a message, a
response grows a field we don't assert, or a remotely-hosted template is updated upstream.
None of these are regressions, yet each turns PRs and nightlies red. `SoftFailFiles`
downgrades a *content diff* in a named golden from a hard failure to a recorded,
non-blocking `SOFTFAIL` marker. It is the last resort — reach for these tools in order:

1. **`Repls` regex mask** — for *describable* volatility (UUIDs, timestamps, versions). Keeps the rest of the file strict.
2. **Field projection** (`<resource>_fields()`, see `.agent/rules/testing.md`) — for "the backend added a field I don't assert". Projecting a response to the fields you assert ignores new fields with zero drift. This fully handles the extra-field case; it should never reach soft-fail.
3. **`SoftFailFiles`** — only for *undescribable* drift you can't regex ahead of time: a reworded message whose new text is unknown, or a wholesale remote-artifact dump we don't control.

### How it works

Add the golden's name to `SoftFailFiles` in `test.toml` (inherited from a parent like any
other config, and surfaced in `out.test.toml` so reviewers see the shield on every PR):

```toml
Badness = "mlops-stacks template is fetched remotely and updated out of band"
SoftFailFiles = ["out.template.txt"]
```

- **`Badness` is required** — every shield carries an in-tree justification.
- **`output.txt` can never be shielded** (a hard config error), and entries must start with `out`. `output.txt` carries the CLI behavior a local regression would corrupt, so a regression in our own logic still turns the test red.
- Only a *content diff* is downgraded. A panic, an unexpected exit code, a missing golden, or an unexpected new file stays a hard failure.

Because the shield is per-file, **route volatile output to its own `out.*` file** and keep
the assertable part strict. Split along a *volatility* seam, not a *convenience* seam:

```bash
# strict — our logic, stays red on any drift
trace $CLI bundle plan | contains.py "Plan: 1 to add"

# volatile — reworded backend output, isolated into its own soft-failed file
trace $CLI some-command-with-drifty-output &> out.drifty.txt
```

A pure local testserver test is deterministic and cannot drift, so soft-fail is only ever
appropriate for cloud tests and tests that consume remotely-fetched artifacts.

### Refresh cadence (oncall)

Soft-failed goldens are reported to the GitHub step summary on every run (via
`tools/softfail_report.py`, which greps the gotestsum JSON for `SOFTFAIL` markers) so drift
is visible even though the build is green. On a cadence, oncall:

1. runs `./task test-update` (cloud tests under the cloud runner),
2. eyeballs `git diff` — mundane drift → commit; anything resembling a real behavior change → investigate,
3. commits the refreshed goldens.

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
