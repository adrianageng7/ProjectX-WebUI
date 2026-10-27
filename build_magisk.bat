@echo off
echo ==============================================
echo   Building ProjectX WebUI Magisk Module
echo ==============================================

echo.
echo [1/4] Building Go Multi-Arch Binaries...
set GOOS=linux
set GOARCH=arm64
go build -ldflags="-s -w" -o projectx_server_arm64 .
copy /Y projectx_server_arm64 magisk_module\bin\projectx_server_arm64
copy /Y projectx_server_arm64 magisk_module\bin\projectx_arm64

set GOARCH=arm
set GOARM=7
go build -ldflags="-s -w" -o projectx_server_arm .
copy /Y projectx_server_arm magisk_module\bin\projectx_server_arm
copy /Y projectx_server_arm magisk_module\bin\projectx_arm

set GOOS=windows
set GOARCH=amd64

echo.
echo [2/4] Building Svelte UI...
cd ui-svelte
call npm run build
cd ..

echo.
echo [3/4] Syncing static assets...
copy /Y magisk_module\static\index.html static\index.html >nul
copy /Y magisk_module\static\clash_generator.html static\clash_generator.html >nul
xcopy /Y /E /I magisk_module\static\assets static\assets >nul

echo.
echo [4/4] Creating UNIX-Compatible ProjectX_Magisk.zip...
go run tools\make_zip\main.go

echo.
echo ==============================================
echo   Build complete! ProjectX_Magisk.zip ready.
echo ==============================================
