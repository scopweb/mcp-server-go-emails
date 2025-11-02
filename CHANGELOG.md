# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Multiple Email Account Support**: Added support for managing multiple email accounts simultaneously
- **Daily Summary Tool**: New `daily_summary` tool that provides consolidated email statistics across all configured accounts
- **JSON Configuration**: Support for `email_config.json` file for configuring multiple email accounts
- **Account Parameter**: Added optional `account` parameter to all existing tools for account-specific operations
- **Default Account**: First account in configuration becomes the default for sending emails when no account is specified

### Changed
- **Go Version**: Updated from Go 1.21 to Go 1.25
- **Configuration System**: Enhanced to support both single account (legacy) and multiple accounts via JSON
- **EmailServer Structure**: Refactored to handle multiple configurations instead of single config
- **Tool Signatures**: All email operations now accept account ID parameter
- **Dependencies**: Updated Go modules for Go 1.25 compatibility

### Fixed
- **Security Tests**: Fixed compilation errors in security test files
- **Path Traversal Detection**: Improved URL-encoded path traversal detection in security tests
- **Test Signatures**: Corrected test function signatures for proper Go testing framework compliance

### Security
- **Enhanced Path Validation**: Improved detection of URL-encoded path traversal attempts
- **Dependency Updates**: Updated to latest secure versions of dependencies
- **Go Security Features**: Leverages Go 1.25 security improvements

### Documentation
- **README Update**: Comprehensive documentation for multiple account configuration
- **Configuration Examples**: Added JSON configuration examples and migration guide
- **Tool Documentation**: Updated tool descriptions with new account parameters

## [1.0.0] - 2024-11-XX

### Added
- Initial MCP Email Server implementation
- Basic email operations: send, read, summarize, delete
- IMAP/SMTP support for Gmail, Outlook, Yahoo
- Claude Desktop integration
- Security testing framework
- Environment variable configuration

### Security
- App Password support for Gmail
- TLS/SSL encryption for all connections
- No credential storage in source code