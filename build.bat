@echo off
echo Building MCP Email Server...
go mod tidy
go build -o email-mcp-server.exe main.go

if exist email-mcp-server.exe (
    echo.
    echo ✓ Build successful! email-mcp-server.exe created
    echo.
    echo Next steps:
    echo 1. Configure your email credentials in .env file
    echo 2. Test connection: set EMAIL_USERNAME=your@email.com ^&^& set EMAIL_PASSWORD=your-password ^&^& go run test_connection.go
    echo 3. Add server to Claude Desktop config with full path to email-mcp-server.exe
    echo 4. Restart Claude Desktop
    echo.
) else (
    echo ✗ Build failed
)