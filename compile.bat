@echo off
echo Compilando MCP Email Server...
cd /d "C:\MCPs\clone\mcp-server-go-emails"

rem Verificar que Go está instalado
go version >nul 2>&1
if errorlevel 1 (
    echo ERROR: Go no está instalado o no está en PATH
    pause
    exit /b 1
)

echo Descargando dependencias...
go mod tidy

echo Compilando...
go build -o email-mcp-server.exe main.go

if exist email-mcp-server.exe (
    echo.
    echo ✓ Compilación exitosa! email-mcp-server.exe creado
    echo Tamaño del archivo:
    dir email-mcp-server.exe | find ".exe"
    echo.
    echo IMPORTANTE: Configura tus credenciales en .env antes de usar
    echo.
) else (
    echo ✗ Error en la compilación
    echo Revisa que Go esté instalado y las dependencias sean correctas
    pause
)
