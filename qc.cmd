@echo off
setlocal
set MODE=%1
if "%MODE%"=="" set MODE=quick

go run .\scripts\check %MODE%
if errorlevel 1 exit /b %errorlevel%