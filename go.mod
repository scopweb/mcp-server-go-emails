module email-mcp-server

go 1.21

require github.com/emersion/go-imap v1.2.1

require modernc.org/sqlite v1.28.0

require (
	github.com/emersion/go-sasl v0.0.0-20220912192320-0145f2c60ead // indirect
	golang.org/x/text v0.3.7 // indirect
)

// Note: Add these dependencies when network is available:
// github.com/robfig/cron/v3 v3.0.1
// go.uber.org/zap v1.26.0
// github.com/google/uuid v1.5.0
