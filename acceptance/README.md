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

## Running acceptance tests on Windows

To run the acceptance tests from a terminal on Windows (eg. Git Bash from VS Code),
you need to install a few prerequisites and optionally make user policy changes.

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
choco install vscode
choco install git
choco install make
choco install jq
choco install python3
choco install uv
choco install go
choco install nodejs
```

### Shim for `python3.exe`

The default Python installation only installs `python.exe` and not `python3.exe`.

We rely on calling `python3` in acceptance tests (shebangs in scripts and elsewhere).

To install `python3` and `pip3` shims for the install, run PowerShell as administrator and execute the following:
```pwsh
# Find your current python.exe
$py = (Get-Command python.exe).Source

# Create a python3.exe shim that points to it
& "$env:ChocolateyInstall\tools\shimgen.exe" `
  --output "$env:ChocolateyInstall\bin\python3.exe" `
  --path   $py

# Optional: pip3, too
$pip = (Get-Command pip.exe).Source
& "$env:ChocolateyInstall\tools\shimgen.exe" `
  --output "$env:ChocolateyInstall\bin\pip3.exe" `
  --path   $pip

refreshenv
python3 --version
pip3 --version
```

### Enable symlink creation

You need to be able to create symlinks.
If you're not an administrator user, enable this by following these steps:

* Press Win+R, type `secpol.msc`, press Enter.
* Go to Local Policies → User Rights Assignment.
* Find "Create symbolic links".
* Add your username to the list.
* Sign out and back in.

### Enable long path support (up to ~32,767 characters)

Some acceptance tests fail if this is not enabled because their paths
exceed the default maximum total length of 260 characters.

* Run "Edit group policy".
* Go to Local Computer Policy → Computer Configuration → Administrative Templates → System → Filesystem → Enable Win32 long paths.
* Enable the setting.
* Reboot.




⚠️ Last resort: Group Policy full disable

If you’re on Windows Pro/Enterprise, and you’ve disabled Tamper Protection, you can permanently disable Defender via Group Policy Editor (gpedit.msc) →
Computer Configuration → Administrative Templates → Windows Components → Microsoft Defender Antivirus → Turn off Microsoft Defender Antivirus → Enabled.
Reboot afterwards.
