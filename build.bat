@echo off
set GOOS=linux
set GOARCH=arm64
echo Building ARM64...
go build -ldflags="-s -w" -o magisk_module/bin/projectx_arm64 .
if %ERRORLEVEL% NEQ 0 exit /b %ERRORLEVEL%
copy /Y magisk_module\bin\projectx_arm64 magisk_module\bin\projectx_server_arm64 >nul

set GOARCH=arm
set GOARM=7
echo Building ARM32...
go build -ldflags="-s -w" -o magisk_module/bin/projectx_arm .
if %ERRORLEVEL% NEQ 0 exit /b %ERRORLEVEL%
copy /Y magisk_module\bin\projectx_arm magisk_module\bin\projectx_server_arm >nul

echo Done!

