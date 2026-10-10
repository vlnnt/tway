@echo off
setlocal

cd /d "%~dp0"

echo Building Tway for Windows...

if not exist "tway-windows" (
    mkdir "tway-windows"
)

set GOOS=windows
set GOARCH=amd64

go build ^
    -trimpath ^
    -ldflags="-H windowsgui -s -w" ^
    -o "tway-windows\tway.exe" ^
    ./cmd/tway

if errorlevel 1 (
    echo.
    echo Build failed.
    exit /b 1
)

copy /Y "assets\tway.ico" "tway-windows\tway.ico" >nul

if errorlevel 1 (
    echo.
    echo Failed to copy icon.
    exit /b 1
)

echo.
echo Build completed:
echo   tway-windows\tway.exe
echo   tway-windows\tway.ico

endlocal