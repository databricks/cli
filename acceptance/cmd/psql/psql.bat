@echo off

REM This script runs psql command bypassing all its arguments to it and exits
REM Test script renames this file to psql.bat in order to capture the arguments that the CLI passes to psql command on Windows

bash "./psql.sh" %*
