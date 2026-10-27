@echo off
:: Script keepalive ADB forward untuk ProjectX WebUI
:: Jalankan script ini lalu buka: http://localhost:8080
echo ========================================
echo  ProjectX ADB Forward Keeper
echo  Phone port 80 ^-^> PC port 8080
echo  Buka browser: http://localhost:8080
echo ========================================
echo.

:loop
adb start-server >nul 2>&1
adb forward tcp:8080 tcp:80 >nul 2>&1
if %errorlevel%==0 (
    echo [%time%] ADB Forward AKTIF: tcp:8080 ^-^> phone:80
) else (
    echo [%time%] WARNING: ADB forward gagal, coba lagi...
)
timeout /t 10 /nobreak >nul
goto loop
