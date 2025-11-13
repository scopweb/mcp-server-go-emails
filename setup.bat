@echo off
setlocal enabledelayedexpansion

echo ================================
echo MCP Email Server - Setup
echo ================================
echo.

REM Step 1: Check prerequisites
echo [1/6] Checking prerequisites...

where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Go is not installed. Please install Go 1.21 or higher.
    echo Visit: https://go.dev/dl/
    exit /b 1
)

for /f "tokens=3" %%i in ('go version') do set GO_VERSION=%%i
echo [OK] Go %GO_VERSION% is installed
echo.

REM Step 2: Create directories
echo [2/6] Creating directories...
if not exist "data" mkdir data
echo [OK] Created data\ directory

if not exist "logs" mkdir logs
echo [OK] Created logs\ directory
echo.

REM Step 3: Copy configuration files
echo [3/6] Setting up configuration files...

if not exist "config\priority_rules.json" (
    copy "config\priority_rules.example.json" "config\priority_rules.json" >nul
    echo [OK] Created config\priority_rules.json
) else (
    echo [SKIP] config\priority_rules.json already exists
)

if not exist "config\ai_config.json" (
    copy "config\ai_config.example.json" "config\ai_config.json" >nul
    echo [OK] Created config\ai_config.json
) else (
    echo [SKIP] config\ai_config.json already exists
)

if not exist ".env" (
    if exist ".env.example" (
        copy ".env.example" ".env" >nul
        echo [OK] Created .env from .env.example
    )
) else (
    echo [SKIP] .env already exists
)
echo.

REM Step 4: Install dependencies
echo [4/6] Installing Go dependencies...
echo [WARNING] This requires internet connectivity!
echo.

ping -n 1 google.com >nul 2>nul
if %ERRORLEVEL% EQU 0 (
    echo [OK] Internet connection detected
    echo.
    echo Running go mod download...
    go mod download
    if %ERRORLEVEL% EQU 0 (
        echo [OK] Dependencies downloaded successfully
    ) else (
        echo [ERROR] Failed to download dependencies
        echo You may need to run 'go mod tidy' manually
        exit /b 1
    )
) else (
    echo [ERROR] No internet connection detected
    echo Cannot download dependencies. Please connect to the internet and run:
    echo   go mod download
    exit /b 1
)
echo.

REM Step 5: Run tests (optional)
echo [5/6] Running tests...
set /p RUN_TESTS="Do you want to run tests? (y/N): "
if /i "%RUN_TESTS%"=="y" (
    echo Running unit tests...
    go test ./test/unit/... -v
    if %ERRORLEVEL% EQU 0 (
        echo [OK] Unit tests passed
    ) else (
        echo [WARNING] Some unit tests failed
    )
    
    echo.
    echo Running integration tests...
    go test ./test/integration/... -v
    if %ERRORLEVEL% EQU 0 (
        echo [OK] Integration tests passed
    ) else (
        echo [WARNING] Some integration tests failed
    )
) else (
    echo [SKIP] Tests skipped
)
echo.

REM Step 6: Build
echo [6/6] Building binary...
go build -o mcp-email-server.exe main.go
if %ERRORLEVEL% EQU 0 (
    echo [OK] Binary built successfully: mcp-email-server.exe
) else (
    echo [ERROR] Build failed
    echo Check the error messages above for details
    exit /b 1
)
echo.

echo ================================
echo Setup Complete!
echo ================================
echo.
echo [OK] All setup steps completed successfully!
echo.
echo Next steps:
echo   1. Configure your email accounts in email_config.json
echo   2. Edit config\priority_rules.json to customize priority rules
echo   3. Run: mcp-email-server.exe
echo.
echo Documentation:
echo   - README.md - General documentation
echo   - INSTALL.md - Detailed installation guide
echo   - docs\USAGE_GUIDE.md - Usage examples
echo   - docs\ARCHITECTURE.md - Technical details
echo.
echo Configuration files:
echo   - email_config.json - Email accounts (create this)
echo   - config\priority_rules.json - Priority rules
echo   - config\ai_config.json - AI configuration
echo   - .env - Environment variables (optional)
echo.
echo Happy emailing! 🚀
echo.
pause
