package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"

	// Intelligent email components
	"email-mcp-server/server"
)

// Load .env file
func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		// Try .env.example if .env doesn't exist
		file, err = os.Open(".env.example")
		if err != nil {
			return // No .env file found, use system environment
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Only set if not already set in system environment
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}
}

// MCP Protocol Types
type MCPRequest struct {
	ID      interface{} `json:"id"` // Can be string, number, or null
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	JSONRPC string      `json:"jsonrpc"`
}

type MCPResponse struct {
	ID      interface{} `json:"id"` // Must match the request ID
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
	JSONRPC string      `json:"jsonrpc"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ToolResult struct {
	Content []TextContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// Email Types
type EmailConfig struct {
	ID          string // Unique identifier for the account
	IMAPHost    string
	IMAPPort    int
	SMTPHost    string
	SMTPPort    int
	Username    string
	Password    string
	UseStartTLS bool
}

type EmailMessage struct {
	ID      uint32    `json:"id"` // Ahora será UID en lugar de SeqNum
	Subject string    `json:"subject"`
	From    string    `json:"from"`
	To      []string  `json:"to"`
	Date    time.Time `json:"date"`
	Body    string    `json:"body"`
	Size    uint32    `json:"size"`
	Flags   []string  `json:"flags"`
}

type EmailSummary struct {
	TotalEmails int           `json:"total_emails"`
	UnreadCount int           `json:"unread_count"`
	RecentCount int           `json:"recent_count"`
	TopSenders  []SenderCount `json:"top_senders"`
	Summary     string        `json:"summary"`
}

type SenderCount struct {
	Email string `json:"email"`
	Count int    `json:"count"`
}

type EmailServer struct {
	configs           []EmailConfig
	defaultAccount    string
	intelligentServer *server.IntelligentEmailServer // AI-powered features
}

func NewEmailServer() *EmailServer {
	// Load .env file first
	loadEnv()

	var configs []EmailConfig
	defaultAccount := ""

	// Try to load from config file first
	if configData, err := os.ReadFile("email_config.json"); err == nil {
		// Load from JSON config file
		var configMap map[string]EmailConfig
		if err := json.Unmarshal(configData, &configMap); err == nil {
			for id, config := range configMap {
				config.ID = id
				configs = append(configs, config)
				if defaultAccount == "" {
					defaultAccount = id
				}
			}
		}
	}

	// If no configs loaded from file, use environment variables for backward compatibility
	if len(configs) == 0 {
		config := EmailConfig{
			ID:          "default",
			IMAPHost:    getEnv("IMAP_HOST", "imap.gmail.com"),
			IMAPPort:    getEnvInt("IMAP_PORT", 993),
			SMTPHost:    getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort:    getEnvInt("SMTP_PORT", 587),
			Username:    getEnv("EMAIL_USERNAME", ""),
			Password:    getEnv("EMAIL_PASSWORD", ""),
			UseStartTLS: getEnv("USE_STARTTLS", "true") == "true",
		}

		if config.Username == "" || config.Password == "" {
			log.Fatal("EMAIL_USERNAME and EMAIL_PASSWORD environment variables are required")
		}

		configs = append(configs, config)
		defaultAccount = "default"
	}

	es := &EmailServer{
		configs:        configs,
		defaultAccount: defaultAccount,
	}

	// Initialize intelligent email server (optional - gracefully fails if config missing)
	dbPath := getEnv("DB_PATH", "./data/emails.db")
	configPath := getEnv("CONFIG_PATH", "./config/priority_rules.json")

	intelligentServer, err := server.NewIntelligentEmailServer(dbPath, configPath)
	if err != nil {
		log.Printf("Warning: Intelligent features disabled (config not found). Basic email features will work normally. Error: %v", err)
	} else {
		es.intelligentServer = intelligentServer
		log.Printf("✅ Intelligent email features enabled (AI classification, priority scoring)")
	}

	return es
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func (es *EmailServer) getConfig(accountID string) (*EmailConfig, error) {
	if accountID == "" {
		accountID = es.defaultAccount
	}
	for _, config := range es.configs {
		if config.ID == accountID {
			return &config, nil
		}
	}
	return nil, fmt.Errorf("account not found: %s", accountID)
}

func (es *EmailServer) connectIMAP(accountID string) (*client.Client, error) {
	config, err := es.getConfig(accountID)
	if err != nil {
		return nil, err
	}

	var c *client.Client
	var connErr error

	if config.IMAPPort == 993 {
		// Use implicit TLS for port 993
		c, connErr = client.DialTLS(fmt.Sprintf("%s:%d", config.IMAPHost, config.IMAPPort), nil)
	} else {
		// Use STARTTLS for other ports
		c, connErr = client.Dial(fmt.Sprintf("%s:%d", config.IMAPHost, config.IMAPPort))
		if connErr != nil {
			return nil, connErr
		}
		if config.UseStartTLS {
			if connErr = c.StartTLS(&tls.Config{ServerName: config.IMAPHost}); connErr != nil {
				return nil, connErr
			}
		}
	}

	if connErr != nil {
		return nil, connErr
	}

	if connErr = c.Login(config.Username, config.Password); connErr != nil {
		c.Close()
		return nil, connErr
	}

	return c, nil
}

func (es *EmailServer) sendEmail(accountID, to, subject, body string) error {
	config, err := es.getConfig(accountID)
	if err != nil {
		return err
	}

	auth := smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		config.Username, to, subject, body)

	addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)
	return smtp.SendMail(addr, auth, config.Username, []string{to}, []byte(msg))
}

func (es *EmailServer) getEmails(accountID string, limit int) ([]EmailMessage, error) {
	c, err := es.connectIMAP(accountID)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	mbox, err := c.Select("INBOX", false)
	if err != nil {
		return nil, err
	}

	if mbox.Messages == 0 {
		return []EmailMessage{}, nil
	}

	from := uint32(1)
	to := mbox.Messages
	if limit > 0 && uint32(limit) < mbox.Messages {
		from = mbox.Messages - uint32(limit) + 1
	}

	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	// CAMBIO CRÍTICO: Incluir UID en el fetch
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchRFC822Size, imap.FetchUid}, messages)
	}()

	var emails []EmailMessage
	for msg := range messages {
		email := EmailMessage{
			ID:      msg.Uid, // CAMBIO: Usar UID en lugar de SeqNum
			Subject: msg.Envelope.Subject,
			From:    formatSingleAddress(msg.Envelope.From),
			To:      formatAddresses(msg.Envelope.To),
			Date:    msg.Envelope.Date,
			Size:    msg.Size,
			Flags:   msg.Flags,
		}

		body := fmt.Sprintf("Subject: %s\nFrom: %s\nDate: %s", // This is not the actual email body
			msg.Envelope.Subject, email.From, msg.Envelope.Date.Format("2006-01-02 15:04:05"))
		email.Body = body

		emails = append(emails, email)
	}

	if err := <-done; err != nil {
		return nil, err
	}

	sort.Slice(emails, func(i, j int) bool {
		return emails[i].Date.After(emails[j].Date)
	})

	return emails, nil
}

func (es *EmailServer) deleteEmail(accountID string, uid uint32) error {
	c, err := es.connectIMAP(accountID)
	if err != nil {
		return err
	}
	defer c.Close()

	if _, err := c.Select("INBOX", false); err != nil {
		return err
	}

	// CAMBIO CRÍTICO: Usar UID set en lugar de sequence set
	uidset := new(imap.SeqSet)
	uidset.AddNum(uid)

	// Marcar como eliminado usando UID STORE
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}

	if err := c.UidStore(uidset, item, flags, nil); err != nil {
		return fmt.Errorf("failed to mark email as deleted: %v", err)
	}

	// Expunge para eliminar permanentemente
	if err := c.Expunge(nil); err != nil {
		return fmt.Errorf("failed to expunge deleted emails: %v", err)
	}

	return nil
}

func (es *EmailServer) summarizeEmails(emails []EmailMessage) EmailSummary {
	unreadCount := 0
	recentCount := 0
	senderMap := make(map[string]int)

	for _, email := range emails {
		// Count unread (no \Seen flag)
		seen := false
		for _, flag := range email.Flags {
			if flag == imap.SeenFlag {
				seen = true
				break
			}
		}
		if !seen {
			unreadCount++
		}

		// Count recent (within 24 hours)
		if time.Since(email.Date) < 24*time.Hour {
			recentCount++
		}

		// Count senders
		senderMap[email.From]++
	}

	// Get top senders
	var topSenders []SenderCount
	for email, count := range senderMap {
		topSenders = append(topSenders, SenderCount{Email: email, Count: count})
	}
	sort.Slice(topSenders, func(i, j int) bool {
		return topSenders[i].Count > topSenders[j].Count
	})
	if len(topSenders) > 5 {
		topSenders = topSenders[:5]
	}

	// Generate summary text
	summary := fmt.Sprintf("Email Summary:\n")
	summary += fmt.Sprintf("• Total: %d emails\n", len(emails))
	summary += fmt.Sprintf("• Unread: %d emails\n", unreadCount)
	summary += fmt.Sprintf("• Recent (24h): %d emails\n", recentCount)

	if len(topSenders) > 0 {
		summary += "\nTop Senders:\n"
		for i, sender := range topSenders {
			if i >= 3 {
				break
			}
			summary += fmt.Sprintf("• %s (%d emails)\n", sender.Email, sender.Count)
		}
	}

	return EmailSummary{
		TotalEmails: len(emails),
		UnreadCount: unreadCount,
		RecentCount: recentCount,
		TopSenders:  topSenders,
		Summary:     summary,
	}
}

func formatSingleAddress(addrs []*imap.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	addr := addrs[0]
	if addr.PersonalName != "" {
		return fmt.Sprintf("%s <%s@%s>", addr.PersonalName, addr.MailboxName, addr.HostName)
	}
	return fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName)
}

func formatAddresses(addrs []*imap.Address) []string {
	var result []string
	for _, addr := range addrs {
		result = append(result, fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName))
	}
	return result
}

// New helper function to extract readable content from raw email
func extractEmailBody(rawEmail string) string {
	lines := strings.Split(rawEmail, "\n")
	inBody := false
	var bodyLines []string

	for _, line := range lines {
		// Headers end with an empty line
		if !inBody && strings.TrimSpace(line) == "" {
			inBody = true
			continue
		}

		if inBody {
			// Skip MIME boundaries and headers within multipart messages
			if strings.HasPrefix(line, "--") ||
				strings.HasPrefix(line, "Content-") ||
				strings.HasPrefix(line, "MIME-Version") {
				continue
			}

			// Clean up the line
			cleanLine := strings.TrimSpace(line)
			if cleanLine != "" {
				bodyLines = append(bodyLines, cleanLine)
			}
		}
	}

	result := strings.Join(bodyLines, "\n")

	// If we got a very short result, return first 500 chars of raw email as fallback
	if len(result) < 10 {
		if len(rawEmail) > 500 {
			return rawEmail[:500] + "..."
		}
		return rawEmail
	}

	return result
}

func main() {
	server := NewEmailServer()
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req MCPRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			log.Printf("Error parsing request: %v", err)
			continue
		}

		if req.JSONRPC != "2.0" {
			log.Printf("Invalid JSON-RPC version: %s", req.JSONRPC)
			continue
		}

		var resp MCPResponse
		resp.ID = req.ID
		resp.JSONRPC = "2.0"

		// Handle notifications (requests without ID)
		if req.ID == nil {
			// For notifications, we don't send a response
			continue
		}

		switch req.Method {
		case "initialize":
			resp.Result = map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": ServerInfo{
					Name:    "email-server",
					Version: "1.0.0",
				},
			}

		case "tools/list":
			resp.Result = map[string]interface{}{
				"tools": []Tool{
					{
						Name:        "get_emails",
						Description: "Get list of emails from inbox",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"account": map[string]interface{}{
									"type":        "string",
									"description": "Account ID to use (optional, uses default if not specified)",
								},
								"limit": map[string]interface{}{
									"type":        "number",
									"description": "Maximum number of emails to retrieve (default: 10)",
									"minimum":     1,
									"maximum":     100,
								},
							},
						},
					},
					{
						Name:        "send_email",
						Description: "Send an email",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"account": map[string]interface{}{
									"type":        "string",
									"description": "Account ID to use for sending (optional, uses default if not specified)",
								},
								"to": map[string]interface{}{
									"type":        "string",
									"description": "Recipient email address",
								},
								"subject": map[string]interface{}{
									"type":        "string",
									"description": "Email subject",
								},
								"body": map[string]interface{}{
									"type":        "string",
									"description": "Email body content",
								},
							},
							"required": []string{"to", "subject", "body"},
						},
					},
					{
						Name:        "summarize_emails",
						Description: "Get a summary of emails in inbox",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"account": map[string]interface{}{
									"type":        "string",
									"description": "Account ID to use (optional, uses default if not specified)",
								},
								"limit": map[string]interface{}{
									"type":        "number",
									"description": "Number of emails to analyze (default: 50)",
									"minimum":     1,
									"maximum":     200,
								},
							},
						},
					},
					{
						Name:        "delete_email",
						Description: "Delete an email by ID",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"account": map[string]interface{}{
									"type":        "string",
									"description": "Account ID to use (optional, uses default if not specified)",
								},
								"id": map[string]interface{}{
									"type":        "number",
									"description": "Email ID to delete",
								},
							},
							"required": []string{"id"},
						},
					},
					{
						Name:        "daily_summary",
						Description: "Get daily summary of emails from all configured accounts",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"limit": map[string]interface{}{
									"type":        "number",
									"description": "Number of emails to analyze per account (default: 50)",
									"minimum":     1,
									"maximum":     200,
								},
							},
						},
					},
					// Intelligent email tools (require configuration)
					{
						Name:        "classify_email",
						Description: "Classify an email into categories (work, personal, promotions, invoice, newsletters, urgent) using intelligent rules",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"from": map[string]interface{}{
									"type":        "string",
									"description": "Email sender address",
								},
								"subject": map[string]interface{}{
									"type":        "string",
									"description": "Email subject",
								},
								"body_snippet": map[string]interface{}{
									"type":        "string",
									"description": "Email body preview (first 500 chars)",
								},
							},
							"required": []string{"from", "subject"},
						},
					},
					{
						Name:        "priority_inbox",
						Description: "Get emails sorted by intelligent priority score (0-100). Returns high-priority emails that need attention",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"account": map[string]interface{}{
									"type":        "string",
									"description": "Account ID to use (optional, uses default if not specified)",
								},
								"min_score": map[string]interface{}{
									"type":        "number",
									"description": "Minimum priority score (0-100, default: 70 for high priority)",
									"minimum":     0,
									"maximum":     100,
								},
								"limit": map[string]interface{}{
									"type":        "number",
									"description": "Maximum number of emails to return (default: 20)",
									"minimum":     1,
									"maximum":     100,
								},
							},
						},
					},
					{
						Name:        "smart_filter",
						Description: "Filter emails using intelligent criteria: category, priority score, sender, date range",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"account": map[string]interface{}{
									"type":        "string",
									"description": "Account ID to use (optional)",
								},
								"category": map[string]interface{}{
									"type":        "string",
									"description": "Filter by category (work, personal, promotions, invoice, newsletters, urgent)",
								},
								"min_priority": map[string]interface{}{
									"type":        "number",
									"description": "Minimum priority score (0-100)",
									"minimum":     0,
									"maximum":     100,
								},
								"unread_only": map[string]interface{}{
									"type":        "boolean",
									"description": "Show only unread emails",
								},
								"limit": map[string]interface{}{
									"type":        "number",
									"description": "Maximum number of emails to return (default: 50)",
									"minimum":     1,
									"maximum":     200,
								},
							},
						},
					},
					{
						Name:        "analyze_priority",
						Description: "Analyze and explain the priority score of an email with detailed reasoning",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"from": map[string]interface{}{
									"type":        "string",
									"description": "Email sender",
								},
								"subject": map[string]interface{}{
									"type":        "string",
									"description": "Email subject",
								},
								"body_snippet": map[string]interface{}{
									"type":        "string",
									"description": "Email body preview",
								},
								"received_at": map[string]interface{}{
									"type":        "string",
									"description": "When email was received (RFC3339 format)",
								},
							},
						},
					},
				},
			}

		case "tools/call":
			if req.Params == nil {
				resp.Error = &MCPError{Code: -32602, Message: "Invalid params: params is required"}
			} else {
				params, ok := req.Params.(map[string]interface{})
				if !ok {
					resp.Error = &MCPError{Code: -32602, Message: "Invalid params: expected object"}
				} else {
					toolParams := ToolCallParams{}
					if name, ok := params["name"].(string); ok {
						toolParams.Name = name
					} else {
						resp.Error = &MCPError{Code: -32602, Message: "Invalid params: name is required"}
					}

					if resp.Error == nil {
						if args, ok := params["arguments"].(map[string]interface{}); ok {
							toolParams.Arguments = args
						} else {
							toolParams.Arguments = make(map[string]interface{})
						}

						result, err := server.handleToolCall(toolParams)
						if err != nil {
							resp.Error = &MCPError{Code: -32603, Message: err.Error()}
						} else {
							resp.Result = result
						}
					}
				}
			}

		default:
			resp.Error = &MCPError{Code: -32601, Message: "Method not found"}
		}

		// Only send response for requests with ID (not notifications)
		if req.ID != nil {
			// CRÍTICO: Asegurar que solo uno de result o error esté presente
			if resp.Error != nil {
				resp.Result = nil
			} else if resp.Result == nil {
				// Si no hay error pero tampoco result, añadir result vacío
				resp.Result = map[string]interface{}{}
			}

			output, err := json.Marshal(resp)
			if err != nil {
				log.Printf("Error marshaling response: %v", err)
				continue
			}

			fmt.Println(string(output))
		}
	}
}

func (es *EmailServer) handleToolCall(params ToolCallParams) (interface{}, error) {
	switch params.Name {
	case "send_email":
		accountID, _ := params.Arguments["account"].(string)
		to, _ := params.Arguments["to"].(string)
		subject, _ := params.Arguments["subject"].(string)
		body, _ := params.Arguments["body"].(string)

		if to == "" || subject == "" || body == "" {
			return nil, fmt.Errorf("missing required parameters: to, subject, body")
		}

		err := es.sendEmail(accountID, to, subject, body)
		if err != nil {
			return nil, fmt.Errorf("failed to send email: %v", err)
		}

		return ToolResult{
			Content: []TextContent{{
				Type: "text",
				Text: fmt.Sprintf("Email sent successfully to %s", to),
			}},
		}, nil

	case "get_emails":
		accountID, _ := params.Arguments["account"].(string)
		limit := 10
		if l, ok := params.Arguments["limit"].(float64); ok {
			limit = int(l)
		}

		emails, err := es.getEmails(accountID, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to get emails: %v", err)
		}

		emailsJSON, _ := json.MarshalIndent(emails, "", "  ")
		return ToolResult{
			Content: []TextContent{{
				Type: "text",
				Text: fmt.Sprintf("Retrieved %d emails:\n\n%s", len(emails), string(emailsJSON)),
			}},
		}, nil

	case "summarize_emails":
		accountID, _ := params.Arguments["account"].(string)
		limit := 50
		if l, ok := params.Arguments["limit"].(float64); ok {
			limit = int(l)
		}

		emails, err := es.getEmails(accountID, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to get emails: %v", err)
		}

		summary := es.summarizeEmails(emails)
		return ToolResult{
			Content: []TextContent{{
				Type: "text",
				Text: summary.Summary,
			}},
		}, nil

	case "delete_email":
		accountID, _ := params.Arguments["account"].(string)
		id, ok := params.Arguments["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid email ID")
		}

		err := es.deleteEmail(accountID, uint32(id))
		if err != nil {
			return nil, fmt.Errorf("failed to delete email: %v", err)
		}

		return ToolResult{
			Content: []TextContent{{
				Type: "text",
				Text: fmt.Sprintf("Email ID %d deleted successfully", uint32(id)),
			}},
		}, nil

	case "daily_summary":
		limit := 50
		if l, ok := params.Arguments["limit"].(float64); ok {
			limit = int(l)
		}

		var allSummaries []string
		totalUnread := 0
		totalRecent := 0

		for _, config := range es.configs {
			emails, err := es.getEmails(config.ID, limit)
			if err != nil {
				allSummaries = append(allSummaries, fmt.Sprintf("❌ Error getting emails for %s: %v", config.ID, err))
				continue
			}

			summary := es.summarizeEmails(emails)
			allSummaries = append(allSummaries, fmt.Sprintf("📧 **Account: %s (%s)**\n%s", config.ID, config.Username, summary.Summary))
			totalUnread += summary.UnreadCount
			totalRecent += summary.RecentCount
		}

		result := fmt.Sprintf("📊 **Daily Email Summary - All Accounts**\n\n")
		result += fmt.Sprintf("📈 **Overall Stats:**\n")
		result += fmt.Sprintf("• Total Unread: %d emails\n", totalUnread)
		result += fmt.Sprintf("• Total Recent (24h): %d emails\n", totalRecent)
		result += fmt.Sprintf("• Accounts monitored: %d\n\n", len(es.configs))

		result += strings.Join(allSummaries, "\n\n")

		return ToolResult{
			Content: []TextContent{{
				Type: "text",
				Text: result,
			}},
		}, nil

	// Intelligent email tools
	case "classify_email":
		if es.intelligentServer == nil {
			return nil, fmt.Errorf("intelligent features not available - configuration file missing")
		}
		result, err := es.intelligentServer.HandleClassifyEmail(params.Arguments)
		if err != nil {
			return nil, fmt.Errorf("classification failed: %v", err)
		}
		return ToolResult{
			Content: []TextContent{{
				Type: "text",
				Text: result,
			}},
		}, nil

	case "priority_inbox":
		if es.intelligentServer == nil {
			return nil, fmt.Errorf("intelligent features not available - configuration file missing")
		}
		result, err := es.intelligentServer.HandlePriorityInbox(params.Arguments)
		if err != nil {
			return nil, fmt.Errorf("priority inbox failed: %v", err)
		}
		return ToolResult{
			Content: []TextContent{{
				Type: "text",
				Text: result,
			}},
		}, nil

	case "smart_filter":
		if es.intelligentServer == nil {
			return nil, fmt.Errorf("intelligent features not available - configuration file missing")
		}
		result, err := es.intelligentServer.HandleSmartFilter(params.Arguments)
		if err != nil {
			return nil, fmt.Errorf("smart filter failed: %v", err)
		}
		return ToolResult{
			Content: []TextContent{{
				Type: "text",
				Text: result,
			}},
		}, nil

	case "analyze_priority":
		if es.intelligentServer == nil {
			return nil, fmt.Errorf("intelligent features not available - configuration file missing")
		}
		result, err := es.intelligentServer.HandleAnalyzePriority(params.Arguments)
		if err != nil {
			return nil, fmt.Errorf("priority analysis failed: %v", err)
		}
		return ToolResult{
			Content: []TextContent{{
				Type: "text",
				Text: result,
			}},
		}, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", params.Name)
	}
}
