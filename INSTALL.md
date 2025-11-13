# 📦 Installation Guide

This guide will help you install and configure the MCP Email Server with all intelligent features.

## ⚠️ Important Notice

**This project requires internet connectivity to install Go dependencies.** The main dependency is `modernc.org/sqlite` which provides SQLite database functionality.

## 🚀 Quick Start

### 1. Prerequisites Check

```bash
# Check Go version (must be 1.21+)
go version

# Check git is installed
git --version

# Check internet connectivity
ping -c 1 google.com
```

### 2. Clone & Setup

```bash
# Clone the repository
git clone <repository-url>
cd mcp-server-go-emails

# Run the setup script
chmod +x setup.sh
./setup.sh
```

The setup script will:
- ✅ Check Go version
- ✅ Create necessary directories
- ✅ Copy configuration files
- ✅ Install dependencies (requires internet)
- ✅ Run tests
- ✅ Build the binary

## 📝 Manual Installation

If the setup script fails, follow these manual steps:

### Step 1: Create Directories

```bash
mkdir -p data
mkdir -p logs
```

### Step 2: Copy Configuration Files

```bash
cp config/priority_rules.example.json config/priority_rules.json
cp config/ai_config.example.json config/ai_config.json
cp .env.example .env
```

### Step 3: Install Dependencies

**⚠️ REQUIRES INTERNET CONNECTION**

```bash
# Download all dependencies
go mod download

# Or install specific dependencies
go get modernc.org/sqlite@v1.28.0
go get github.com/emersion/go-imap@v1.2.1
```

### Step 4: Configure Email Accounts

Edit `email_config.json` or `.env` with your email credentials:

```json
{
  "personal": {
    "IMAPHost": "imap.gmail.com",
    "IMAPPort": 993,
    "SMTPHost": "smtp.gmail.com",
    "SMTPPort": 587,
    "Username": "your-email@gmail.com",
    "Password": "your-app-password",
    "UseStartTLS": true
  }
}
```

**For Gmail users:**
1. Enable 2FA: https://myaccount.google.com/security
2. Generate App Password: https://myaccount.google.com/apppasswords
3. Use the 16-character app password

### Step 5: Build

```bash
go build -o mcp-email-server main.go
```

### Step 6: Test

```bash
# Unit tests
go test ./test/unit/... -v

# Integration tests
go test ./test/integration/... -v

# All tests
go test ./... -v
```

### Step 7: Run

```bash
./mcp-email-server
```

## 🔧 Troubleshooting

### Error: "missing go.sum entry for module"

**Problem:** Go modules are not downloaded.

**Solution:**
```bash
go mod tidy
go mod download
```

If you still get errors, delete `go.sum` and try again:
```bash
rm go.sum
go mod tidy
```

### Error: "dial tcp: lookup storage.googleapis.com"

**Problem:** No internet connectivity.

**Solution:**
1. Connect to the internet
2. Retry: `go mod download`
3. If behind a proxy, configure Go proxy:
   ```bash
   go env -w GOPROXY=https://proxy.golang.org,direct
   ```

### Error: "database is locked"

**Problem:** Another instance is using the database.

**Solution:**
```bash
# Kill all instances
pkill -9 mcp-email-server

# Remove lock file
rm data/emails.db-wal
rm data/emails.db-shm
```

### Error: "Authentication Failed" (Email)

**Problem:** Wrong credentials or app password not enabled.

**Solution:**
- For Gmail: Use App Password, not regular password
- For Outlook: Enable IMAP in settings
- For Yahoo: Generate App Password if using 2FA

### Build Errors

**Problem:** Go version too old.

**Solution:**
```bash
# Update Go to 1.21+
# Visit: https://go.dev/dl/
```

## 🐧 Platform-Specific Instructions

### Linux

```bash
# Install Go (if not installed)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Build
go build -o mcp-email-server main.go

# Make executable
chmod +x mcp-email-server

# Run
./mcp-email-server
```

### macOS

```bash
# Install Go (if not installed)
brew install go

# Build
go build -o mcp-email-server main.go

# Make executable
chmod +x mcp-email-server

# Run
./mcp-email-server
```

### Windows

```powershell
# Build
go build -o mcp-email-server.exe main.go

# Run
.\mcp-email-server.exe
```

## 📋 Verification Checklist

After installation, verify everything works:

- [ ] Go version is 1.21 or higher
- [ ] Dependencies downloaded successfully
- [ ] `data/` directory exists
- [ ] Configuration files exist in `config/`
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Binary builds successfully
- [ ] Email credentials configured
- [ ] Server starts without errors

## 🔐 Security Notes

1. **Never commit `.env` or `email_config.json`** to git
2. Use **App Passwords**, not regular passwords
3. Keep **credentials secure** (chmod 600 on config files)
4. Enable **2FA** on all email accounts
5. Use **TLS/SSL** for all connections (default)

## 📚 Next Steps

After successful installation:

1. **Configure Priority Rules**: Edit `config/priority_rules.json`
2. **Set Up Claude Desktop**: Add MCP server to Claude config
3. **Read Usage Guide**: Check `docs/USAGE_GUIDE.md`
4. **Run First Tests**: Try the intelligent tools

## 🆘 Getting Help

If you encounter issues:

1. Check this guide's troubleshooting section
2. Review `README.md` for configuration details
3. Check logs in `logs/` directory
4. Open an issue on GitHub with:
   - Go version (`go version`)
   - Error message
   - Steps to reproduce

## 📦 Dependencies

This project requires:

- **Runtime Dependencies:**
  - `modernc.org/sqlite v1.28.0` - SQLite database (pure Go)
  - `github.com/emersion/go-imap v1.2.1` - IMAP client

- **Future Dependencies (coming soon):**
  - `github.com/robfig/cron/v3` - Task scheduling
  - `go.uber.org/zap` - Logging
  - `github.com/google/uuid` - UUID generation

All dependencies are managed via `go.mod` and can be installed with:
```bash
go mod download
```

## 🎯 Installation Complete!

Once everything is set up, you should have:

```
mcp-server-go-emails/
├── mcp-email-server          # Built binary
├── data/                      # Database directory
├── config/
│   ├── priority_rules.json    # Priority rules
│   └── ai_config.json         # AI configuration
├── email_config.json          # Email accounts (you create this)
└── .env                       # Environment variables (optional)
```

You're ready to use the MCP Email Server! 🎉

See [README.md](README.md) for usage instructions and [docs/USAGE_GUIDE.md](docs/USAGE_GUIDE.md) for detailed examples.
