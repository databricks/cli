@echo off
echo %~n0 %*>>commands.log
if "%*"=="%VALIDATION_TEST_UPDATE%" (
    copy /y "%VALIDATION_TEST_PACKAGE_JSON%" package.json >nul || exit /b 1
)
if "%*"=="%VALIDATION_TEST_FAIL%" (
    echo validation stdout
    echo validation stderr 1>&2
    exit /b 17
)
