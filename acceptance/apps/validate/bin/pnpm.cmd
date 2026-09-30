@echo off
echo pnpm %*>>commands.log
if "%*"=="%VALIDATION_TEST_FAIL%" (
    echo validation stdout
    echo validation stderr 1>&2
    exit /b 17
)
