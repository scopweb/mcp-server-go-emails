@echo off
echo Loading environment from .env file...

REM Read .env file and set variables
for /f "usebackq tokens=1,2 delims==" %%a in (".env.example") do (
    if not "%%a"=="" if not "%%a"=="REM" if not "%%a"=="#" (
        set %%a=%%b
        echo Set %%a=%%b
    )
)

echo.
echo Starting email MCP server...
email-mcp-server.exe