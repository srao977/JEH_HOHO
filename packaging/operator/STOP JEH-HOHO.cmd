@echo off
rem File: STOP JEH-HOHO.cmd
rem Date: 2026-09-17
rem Product: JEH-HOHO
rem Purpose: Request clean shutdown from this package's owning controller.
cd /d "%~dp0"
"%~dp0bin\jeh-hoho.exe" stop --home "%~dp0"
pause