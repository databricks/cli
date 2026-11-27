@echo off
echo %~n0 %*>>commands.log
if "%*"=="%VALIDATION_TEST_FAIL%" (
    echo validation stdout
    1>&2 echo %VALIDATION_TEST_STDERR%
    exit /b 17
)
