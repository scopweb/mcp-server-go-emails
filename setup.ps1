# MCP Email Server - Setup Script for Windows (PowerShell)
# Run this with: powershell -ExecutionPolicy Bypass -File setup.ps1

$ErrorActionPreference = "Stop"

# Colors for output
function Write-Info {
    param([string]$Message)
    Write-Host "ℹ️  $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠️  $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
}

function Write-Header {
    param([string]$Message)
    Write-Host ""
    Write-Host "================================" -ForegroundColor Blue
    Write-Host $Message -ForegroundColor Blue
    Write-Host "================================" -ForegroundColor Blue
    Write-Host ""
}

# Main setup function
function Main {
    Write-Header "MCP Email Server - Setup"

    # Step 1: Check prerequisites
    Write-Info "Step 1: Checking prerequisites..."
    
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Error "Go is not installed. Please install Go 1.21 or higher."
        Write-Info "Visit: https://go.dev/dl/"
        exit 1
    }

    $goVersion = (go version) -replace ".*go(\d+\.\d+\.\d+).*", '$1'
    Write-Success "Go $goVersion is installed"

    if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
        Write-Warning "Git is not installed. Some features may not work."
    } else {
        Write-Success "Git is installed"
    }
    Write-Host ""

    # Step 2: Create directories
    Write-Info "Step 2: Creating directories..."
    
    if (-not (Test-Path "data")) {
        New-Item -ItemType Directory -Path "data" | Out-Null
        Write-Success "Created data\ directory"
    } else {
        Write-Warning "data\ directory already exists"
    }
    
    if (-not (Test-Path "logs")) {
        New-Item -ItemType Directory -Path "logs" | Out-Null
        Write-Success "Created logs\ directory"
    } else {
        Write-Warning "logs\ directory already exists"
    }
    Write-Host ""

    # Step 3: Copy configuration files
    Write-Info "Step 3: Setting up configuration files..."
    
    if (-not (Test-Path "config\priority_rules.json")) {
        Copy-Item "config\priority_rules.example.json" "config\priority_rules.json"
        Write-Success "Created config\priority_rules.json"
    } else {
        Write-Warning "config\priority_rules.json already exists, skipping..."
    }

    if (-not (Test-Path "config\ai_config.json")) {
        Copy-Item "config\ai_config.example.json" "config\ai_config.json"
        Write-Success "Created config\ai_config.json"
    } else {
        Write-Warning "config\ai_config.json already exists, skipping..."
    }

    if (-not (Test-Path ".env")) {
        if (Test-Path ".env.example") {
            Copy-Item ".env.example" ".env"
            Write-Success "Created .env from .env.example"
        }
    } else {
        Write-Warning ".env already exists, skipping..."
    }
    Write-Host ""

    # Step 4: Install dependencies
    Write-Info "Step 4: Installing Go dependencies..."
    Write-Warning "This requires internet connectivity!"
    
    try {
        $ping = Test-Connection -ComputerName google.com -Count 1 -Quiet
        if ($ping) {
            Write-Success "Internet connection detected"
            
            Write-Info "Running go mod download..."
            go mod download
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Dependencies downloaded successfully"
            } else {
                Write-Error "Failed to download dependencies"
                Write-Info "You may need to run 'go mod tidy' manually"
                exit 1
            }
        } else {
            Write-Error "No internet connection detected"
            Write-Warning "Cannot download dependencies. Please connect to the internet and run:"
            Write-Info "  go mod download"
            exit 1
        }
    } catch {
        Write-Warning "Could not verify internet connection"
    }
    Write-Host ""

    # Step 5: Run tests (optional)
    Write-Info "Step 5: Running tests..."
    
    $runTests = Read-Host "Do you want to run tests? (y/N)"
    if ($runTests -eq "y" -or $runTests -eq "Y") {
        Write-Info "Running unit tests..."
        go test ./test/unit/... -v
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Unit tests passed"
        } else {
            Write-Warning "Some unit tests failed (this may be normal without email config)"
        }

        Write-Host ""
        Write-Info "Running integration tests..."
        go test ./test/integration/... -v
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Integration tests passed"
        } else {
            Write-Warning "Some integration tests failed (this may be normal without email config)"
        }
    } else {
        Write-Info "Skipping tests"
    }
    Write-Host ""

    # Step 6: Build
    Write-Info "Step 6: Building binary..."
    
    go build -o mcp-email-server.exe main.go
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Binary built successfully: mcp-email-server.exe"
    } else {
        Write-Error "Build failed"
        Write-Info "Check the error messages above for details"
        exit 1
    }
    Write-Host ""

    # Summary
    Write-Header "Setup Complete!"
    
    Write-Host "✅ All setup steps completed successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "📋 Next steps:"
    Write-Host "   1. Configure your email accounts in email_config.json"
    Write-Host "   2. Edit config\priority_rules.json to customize priority rules"
    Write-Host "   3. Run: .\mcp-email-server.exe"
    Write-Host ""
    Write-Host "📚 Documentation:"
    Write-Host "   - README.md - General documentation"
    Write-Host "   - INSTALL.md - Detailed installation guide"
    Write-Host "   - docs\USAGE_GUIDE.md - Usage examples"
    Write-Host "   - docs\ARCHITECTURE.md - Technical details"
    Write-Host ""
    Write-Host "🔧 Configuration files:"
    Write-Host "   - email_config.json - Email accounts (create this)"
    Write-Host "   - config\priority_rules.json - Priority rules"
    Write-Host "   - config\ai_config.json - AI configuration"
    Write-Host "   - .env - Environment variables (optional)"
    Write-Host ""
    Write-Success "Happy emailing! 🚀"
    Write-Host ""
}

# Run main function
Main
