@echo off
rem File: CREATE SUPPORT REPORT.cmd
rem Date: 2026-09-17
rem Product: JEH-HOHO
rem Purpose: Create a sanitized package-local support report.
cd /d "%~dp0"
"%~dp0bin\jeh-hoho.exe" support --home "%~dp0"
pause