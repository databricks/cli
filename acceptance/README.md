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

Any file starting with "LOG" will be logged to test log (visible with go test -v).

See [selftest](./selftest) for more features.

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
