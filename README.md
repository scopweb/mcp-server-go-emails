# MCP Email Server

A Model Context Protocol (MCP) server implemented in Go for email management via IMAP/SMTP. Supports multiple email accounts and designed for integration with Claude Desktop.

## Features

- Send emails via SMTP from multiple accounts
- Read emails from IMAP servers for multiple accounts
- Generate inbox summaries per account or across all accounts
- Delete specific emails from any account
- Daily summary across all configured accounts
- Gmail, Outlook, Yahoo support with App Passwords
- Full MCP protocol compliance
- JSON configuration for multiple accounts

## Installation

### Prerequisites

- Go 1.25 or higher
- Email accounts with IMAP/SMTP access
- Claude Desktop

### Setup

1. Clone or download this repository
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Build the server:
   ```bash
   go build -o email-mcp-server.exe main.go
   ```

## Configuration

### Single Account (Legacy Mode)

For backward compatibility, you can still use environment variables for a single account:

```env
EMAIL_USERNAME=your-email@gmail.com
EMAIL_PASSWORD=your-app-password
IMAP_HOST=imap.gmail.com
IMAP_PORT=993
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
USE_STARTTLS=true
```

### Multiple Accounts (Recommended)

Create an `email_config.json` file in the same directory as the executable. Use `email_config.example.json` as a template:

```json
{
  "personal": {
    "IMAPHost": "imap.gmail.com",
    "IMAPPort": 993,
    "SMTPHost": "smtp.gmail.com",
    "SMTPPort": 587,
    "Username": "yourpersonal@gmail.com",
    "Password": "your_app_password_here",
    "UseStartTLS": true
  },
  "work": {
    "IMAPHost": "outlook.office365.com",
    "IMAPPort": 993,
    "SMTPHost": "smtp-mail.outlook.com",
    "SMTPPort": 587,
    "Username": "yourwork@company.com",
    "Password": "your_app_password_here",
    "UseStartTLS": true
  },
  "secondary": {
    "IMAPHost": "imap.mail.yahoo.com",
    "IMAPPort": 993,
    "SMTPHost": "smtp.mail.yahoo.com",
    "SMTPPort": 587,
    "Username": "yoursecondary@yahoo.com",
    "Password": "your_app_password_here",
    "UseStartTLS": true
  }
}
```

#### How Default Account Works

**The first account in the JSON object becomes the default account for sending emails.**

In the example above:
- `"personal"` is the default account (first in the JSON)
- When you use `send_email` without specifying an account, it uses the personal account
- You can override this by specifying `account: "work"` in your requests

#### Account Configuration Fields

Each account in the JSON requires these fields:

- `IMAPHost`: IMAP server hostname (e.g., "imap.gmail.com")
- `IMAPPort`: IMAP server port (usually 993 for SSL)
- `SMTPHost`: SMTP server hostname (e.g., "smtp.gmail.com")
- `SMTPPort`: SMTP server port (usually 587 for STARTTLS)
- `Username`: Your email address
- `Password`: App password (not regular password)
- `UseStartTLS`: `true` for most providers, enables secure connection upgrade

### Email Provider Setup

#### Gmail Setup
1. Enable 2-Factor Authentication in your Google Account
2. Go to [Google App Passwords](https://myaccount.google.com/apppasswords)
3. Generate a 16-character App Password
4. Use this App Password in the `Password` field (not your regular password)

#### Outlook/Office 365 Setup
1. Use your regular password or generate an App Password
2. Ensure IMAP is enabled in Outlook settings
3. Use `outlook.office365.com` for IMAP and `smtp-mail.outlook.com` for SMTP

#### Yahoo Setup
1. Enable IMAP in Yahoo Mail settings
2. Generate an App Password if you have 2FA enabled
3. Use `imap.mail.yahoo.com` for IMAP and `smtp.mail.yahoo.com` for SMTP

### Claude Desktop Configuration

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "email": {
      "command": "C:\\path\\to\\email-mcp-server.exe",
      "args": [],
      "env": {}
    }
  }
}
```

**Note**: No environment variables needed when using JSON configuration.

## Usage

Once configured with Claude Desktop, you can use natural language commands:

- "Send an email to john@example.com about the meeting tomorrow"
- "Send an email from my work account to boss@company.com"
- "What new emails do I have in my personal account?"
- "Summarize my work inbox"
- "Show me the daily summary from all accounts"
- "Delete email with ID 5 from personal account"

## Available Tools

### send_email
Send an email to a recipient
- `account`: Account ID to use (optional, uses default if not specified)
- `to`: Recipient email address
- `subject`: Email subject
- `body`: Email content

### get_emails
Retrieve recent emails from inbox
- `account`: Account ID to use (optional, uses default if not specified)
- `limit`: Maximum number of emails (default: 10)

### summarize_emails
Generate inbox summary with statistics
- `account`: Account ID to use (optional, uses default if not specified)
- `limit`: Number of emails to analyze (default: 50)

### delete_email
Delete a specific email
- `account`: Account ID to use (optional, uses default if not specified)
- `id`: Email ID to delete

### daily_summary
Generate daily summary across all configured accounts
- `limit`: Number of emails to analyze per account (default: 50)

## Account Management

### Default Account Behavior

- **Default Account**: The first account defined in `email_config.json` is automatically set as the default
- **Sending Emails**: When no `account` parameter is specified in `send_email`, the default account is used
- **Reading Emails**: When no `account` parameter is specified in other tools, the default account is used
- **Changing Default**: Reorder accounts in JSON or rename the first account key

### Account IDs

Account IDs are the keys you define in your JSON configuration:
```json
{
  "personal": { ... },  // Account ID: "personal"
  "work": { ... },      // Account ID: "work"
  "backup": { ... }     // Account ID: "backup"
}
```

### Using Accounts in Commands

#### Automatic (Default Account)
```
"Send an email to john@example.com" → Uses default account
"Show me my emails" → Uses default account
"Summarize my inbox" → Uses default account
```

#### Explicit Account Specification
```
"Send an email from my work account to boss@company.com"
"Show me emails from personal account"
"Get summary of work inbox"
"Delete email 123 from backup account"
```

#### Cross-Account Operations
```
"Show me the daily summary from all accounts" → Uses daily_summary tool
```

### Best Practices

1. **Put your most-used account first** in the JSON to make it the default
2. **Use descriptive account names** like "personal", "work", "gmail", "outlook"
3. **Test each account** after configuration using account-specific commands
4. **Use daily_summary** regularly to monitor all accounts at once

## Supported Email Providers

- Gmail (imap.gmail.com, smtp.gmail.com)
- Outlook/Office 365 (outlook.office365.com, smtp-mail.outlook.com)
- Yahoo (imap.mail.yahoo.com, smtp.mail.yahoo.com)
- Any standard IMAP/SMTP server

## Testing

Test your connection before using with Claude:

```bash
# For single account
set EMAIL_USERNAME=your@gmail.com
set EMAIL_PASSWORD=your-app-password
go run test_connection.go

# For multiple accounts, create email_config.json first
go test ./test/security -v
```

## Security

- Uses App Passwords instead of main account passwords
- All connections use TLS/SSL encryption
- Credentials stored in JSON config file
- No credential storage in source code
- Environment variables supported for single account setup

## Troubleshooting

### Authentication Issues
**"Authentication Failed"**
- Use App Password, not regular password for Gmail
- Enable 2FA for Gmail accounts
- Verify credentials are correct in `email_config.json`
- Test each account individually

### Connection Issues
**"Connection Refused"**
- Check host/port settings for your email provider
- Verify firewall allows outbound connections on ports 993 (IMAP) and 587 (SMTP)
- Try different ports if needed (465 for SMTP SSL)

### Account Configuration Issues
**"Account Not Found"**
- Verify account ID matches exactly (case-sensitive) with your JSON config keys
- Check JSON syntax is valid (use a JSON validator)
- Ensure `email_config.json` is in the same directory as the executable

**"Default account not working as expected"**
- The first account in JSON becomes default
- Reorder accounts in JSON to change default
- Specify account explicitly if needed

### Multiple Accounts Issues
**"Multiple accounts not working"**
- Ensure `email_config.json` exists and is readable
- Check file permissions
- Verify JSON format is correct
- Test with `go test ./test/security -v` to ensure basic functionality

**"Wrong account being used"**
- Check which account is first in your JSON file
- Use explicit `account` parameter in requests
- Verify account IDs are unique

### General Issues
**"Server Not Found"**
- Verify the executable path in Claude Desktop configuration
- Check if the executable was built successfully
- Ensure Go version is 1.25 or higher

**"Tools not appearing in Claude"**
- Restart Claude Desktop after configuration changes
- Check Claude Desktop logs for errors
- Verify MCP server is running (check Windows Task Manager)

### Testing Your Setup

```bash
# Test security functionality
go test ./test/security -v

# Build the executable
go build -o email-mcp-server.exe main.go

# Test with sample config (create email_config.json first)
./email-mcp-server.exe
```

## Migration from Single to Multiple Accounts

### Step-by-Step Migration

1. **Backup your current setup**
   ```bash
   # Save your current environment variables
   echo "EMAIL_USERNAME=$EMAIL_USERNAME" > backup.env
   echo "EMAIL_PASSWORD=$EMAIL_PASSWORD" >> backup.env
   # ... save other env vars
   ```

2. **Create email_config.json**
   ```json
   {
     "default": {
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

3. **Test the new configuration**
   ```bash
   # Remove environment variables temporarily
   # Test with: go test ./test/security -v
   ```

4. **Update Claude Desktop config**
   - Remove `env` section from `claude_desktop_config.json`
   - Restart Claude Desktop

5. **Add additional accounts**
   ```json
   {
     "personal": {
       "IMAPHost": "imap.gmail.com",
       "IMAPPort": 993,
       "SMTPHost": "smtp.gmail.com",
       "SMTPPort": 587,
       "Username": "personal@gmail.com",
       "Password": "app-password-1",
       "UseStartTLS": true
     },
     "work": {
       "IMAPHost": "outlook.office365.com",
       "IMAPPort": 993,
       "SMTPHost": "smtp-mail.outlook.com",
       "SMTPPort": 587,
       "Username": "work@company.com",
       "Password": "password-2",
       "UseStartTLS": true
     }
   }
   ```

6. **Verify all accounts work**
   - Test `daily_summary` to ensure all accounts are accessible
   - Test sending from different accounts
   - Test reading from different accounts

### Rollback Plan

If something goes wrong:
1. Restore `email_config.json.backup`
2. Restore environment variables from `backup.env`
3. Update Claude Desktop config to include env vars again
4. Restart Claude Desktop

### Benefits of Multiple Accounts

- **Unified Management**: Control all email accounts from one interface
- **Daily Overview**: Get consolidated summaries across accounts
- **Account Separation**: Keep personal and work emails organized
- **Flexible Sending**: Choose which account to send from
- **Scalability**: Easy to add/remove accounts by editing JSON

## ⚠️ Important Notice

**This project is for educational and testing purposes only.**

- This software is provided "as is" without warranty of any kind
- The authors and contributors are not responsible for any misuse of this software
- Users are responsible for complying with email provider terms of service and applicable laws
- Use App Passwords and follow security best practices when configuring email accounts
- Do not use this software for spam, harassment, or any illegal activities
- Test thoroughly in development environments before production use

## License

MIT License

## Contributing

Pull requests and issues are welcome.