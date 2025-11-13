#!/bin/bash

# MCP Email Server - Setup Script
# This script automates the installation and configuration process

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print functions
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_header() {
    echo ""
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================${NC}"
    echo ""
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Main setup function
main() {
    print_header "MCP Email Server - Setup"

    # Step 1: Check prerequisites
    print_info "Step 1: Checking prerequisites..."
    
    if ! command_exists go; then
        print_error "Go is not installed. Please install Go 1.21 or higher."
        print_info "Visit: https://go.dev/dl/"
        exit 1
    fi

    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_success "Go $GO_VERSION is installed"

    if ! command_exists git; then
        print_warning "Git is not installed. Some features may not work."
    else
        print_success "Git is installed"
    fi

    # Step 2: Create directories
    print_info "Step 2: Creating directories..."
    
    mkdir -p data
    print_success "Created data/ directory"
    
    mkdir -p logs
    print_success "Created logs/ directory"

    # Step 3: Copy configuration files
    print_info "Step 3: Setting up configuration files..."
    
    if [ ! -f config/priority_rules.json ]; then
        cp config/priority_rules.example.json config/priority_rules.json
        print_success "Created config/priority_rules.json"
    else
        print_warning "config/priority_rules.json already exists, skipping..."
    fi

    if [ ! -f config/ai_config.json ]; then
        cp config/ai_config.example.json config/ai_config.json
        print_success "Created config/ai_config.json"
    else
        print_warning "config/ai_config.json already exists, skipping..."
    fi

    if [ ! -f .env ]; then
        if [ -f .env.example ]; then
            cp .env.example .env
            print_success "Created .env from .env.example"
        fi
    else
        print_warning ".env already exists, skipping..."
    fi

    # Step 4: Install dependencies
    print_info "Step 4: Installing Go dependencies..."
    print_warning "This requires internet connectivity!"
    
    if ping -c 1 google.com &> /dev/null; then
        print_success "Internet connection detected"
        
        print_info "Running go mod download..."
        if go mod download; then
            print_success "Dependencies downloaded successfully"
        else
            print_error "Failed to download dependencies"
            print_info "You may need to run 'go mod tidy' manually"
            exit 1
        fi
    else
        print_error "No internet connection detected"
        print_warning "Cannot download dependencies. Please connect to the internet and run:"
        print_info "  go mod download"
        exit 1
    fi

    # Step 5: Run tests (optional)
    print_info "Step 5: Running tests..."
    
    read -p "Do you want to run tests? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "Running unit tests..."
        if go test ./test/unit/... -v; then
            print_success "Unit tests passed"
        else
            print_warning "Some unit tests failed (this may be normal without email config)"
        fi

        print_info "Running integration tests..."
        if go test ./test/integration/... -v; then
            print_success "Integration tests passed"
        else
            print_warning "Some integration tests failed (this may be normal without email config)"
        fi
    else
        print_info "Skipping tests"
    fi

    # Step 6: Build
    print_info "Step 6: Building binary..."
    
    if go build -o mcp-email-server main.go; then
        chmod +x mcp-email-server
        print_success "Binary built successfully: ./mcp-email-server"
    else
        print_error "Build failed"
        print_info "Check the error messages above for details"
        exit 1
    fi

    # Step 7: Summary
    print_header "Setup Complete!"
    
    echo "✅ All setup steps completed successfully!"
    echo ""
    echo "📋 Next steps:"
    echo "   1. Configure your email accounts in email_config.json"
    echo "   2. Edit config/priority_rules.json to customize priority rules"
    echo "   3. Run: ./mcp-email-server"
    echo ""
    echo "📚 Documentation:"
    echo "   - README.md - General documentation"
    echo "   - INSTALL.md - Detailed installation guide"
    echo "   - docs/USAGE_GUIDE.md - Usage examples"
    echo "   - docs/ARCHITECTURE.md - Technical details"
    echo ""
    echo "🔧 Configuration files:"
    echo "   - email_config.json - Email accounts (create this)"
    echo "   - config/priority_rules.json - Priority rules"
    echo "   - config/ai_config.json - AI configuration"
    echo "   - .env - Environment variables (optional)"
    echo ""
    print_success "Happy emailing! 🚀"
}

# Run main function
main "$@"
