@echo off
rem File: STATUS JEH-HOHO.cmd
rem Date: 2026-09-17
rem Product: JEH-HOHO
rem Purpose: Show plain current product status and the Viewer address.
cd /d "%~dp0"
"%~dp0bin\jeh-hoho.exe" status --home "%~dp0"
pause