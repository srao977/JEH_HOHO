@echo off
rem File: START JEH-HOHO.cmd
rem Date: 2026-09-17
rem Product: JEH-HOHO
rem Purpose: Start the complete packaged product with one operator action.
cd /d "%~dp0"
"%~dp0bin\jeh-hoho.exe" start --home "%~dp0"
if errorlevel 1 pause