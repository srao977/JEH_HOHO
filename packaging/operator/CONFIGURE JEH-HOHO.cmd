@echo off
rem File: CONFIGURE JEH-HOHO.cmd
rem Date: 2026-09-17
rem Product: JEH-HOHO
rem Purpose: Create and open the protected first-use Alpaca credential file.
cd /d "%~dp0"
if not exist "config\credentials.json" copy /y "config\credentials.example.json" "config\credentials.json" >nul
notepad "config\credentials.json"