package server

import (
	"email-mcp-server/ai"
	"email-mcp-server/config"
	"email-mcp-server/storage"
	"encoding/json"
	"fmt"
	"time"
)

// MCPToolDefinitions returns all MCP tool definitions including new intelligent tools
func GetIntelligentTools() []Tool {
	return []Tool{
		{
			Name:        "classify_email",
			Description: "Classify an email into categories (work, personal, promotions, invoice, newsletters, urgent) using intelligent rules",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"account": map[string]interface{}{
						"type":        "string",
						"description": "Account ID to use (optional)",
					},
					"email_id": map[string]interface{}{
						"type":        "string",
						"description": "Email ID to classify (if analyzing from database)",
					},
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
						"enum":        []string{"work", "personal", "promotions", "invoice", "newsletters", "urgent"},
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
					"date_from": map[string]interface{}{
						"type":        "string",
						"description": "Filter emails from this date (YYYY-MM-DD or RFC3339)",
					},
					"date_to": map[string]interface{}{
						"type":        "string",
						"description": "Filter emails until this date (YYYY-MM-DD or RFC3339)",
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
					"account": map[string]interface{}{
						"type":        "string",
						"description": "Account ID",
					},
					"email_id": map[string]interface{}{
						"type":        "string",
						"description": "Email ID to analyze",
					},
					"from": map[string]interface{}{
						"type":        "string",
						"description": "Email sender (if not analyzing from database)",
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
	}
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// IntelligentEmailServer extends EmailServer with AI capabilities
type IntelligentEmailServer struct {
	db         *storage.Database
	classifier *ai.Classifier
	priority   *ai.PriorityEngine
	config     *config.PriorityConfig
}

// NewIntelligentEmailServer creates a new intelligent email server
func NewIntelligentEmailServer(dbPath, configPath string) (*IntelligentEmailServer, error) {
	// Initialize database
	dbConfig := storage.DefaultConfig()
	dbConfig.Path = dbPath
	db, err := storage.New(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Load configuration
	cfg, err := config.LoadPriorityConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize AI components
	classifier := ai.NewClassifier(cfg, db)
	priorityEngine := ai.NewPriorityEngine(cfg, db, classifier)

	return &IntelligentEmailServer{
		db:         db,
		classifier: classifier,
		priority:   priorityEngine,
		config:     cfg,
	}, nil
}

// HandleClassifyEmail handles the classify_email tool
func (ies *IntelligentEmailServer) HandleClassifyEmail(args map[string]interface{}) (string, error) {
	// Extract parameters
	emailID, _ := args["email_id"].(string)
	from, _ := args["from"].(string)
	subject, _ := args["subject"].(string)
	bodySnippet, _ := args["body_snippet"].(string)

	if from == "" || subject == "" {
		return "", fmt.Errorf("from and subject are required")
	}

	// Create email object
	email := &ai.Email{
		ID:          emailID,
		From:        from,
		Subject:     subject,
		BodySnippet: bodySnippet,
		ReceivedAt:  time.Now(),
	}

	// Classify
	result, err := ies.classifier.Classify(email)
	if err != nil {
		return "", fmt.Errorf("classification failed: %w", err)
	}

	// Save to database if email_id provided
	if emailID != "" {
		if err := ies.classifier.SaveClassification(result); err != nil {
			// Non-fatal, just log
			fmt.Printf("Warning: failed to save classification: %v\n", err)
		}
	}

	// Format response
	response := fmt.Sprintf(`📧 Email Classification Result

Category: %s
Confidence: %.0f%%
Method: %s
Tags: %v

Reasoning: %s

From: %s
Subject: %s`,
		result.Category,
		result.Confidence*100,
		result.Method,
		result.Tags,
		result.Reasoning,
		from,
		subject,
	)

	return response, nil
}

// HandlePriorityInbox handles the priority_inbox tool
func (ies *IntelligentEmailServer) HandlePriorityInbox(args map[string]interface{}) (string, error) {
	accountID, _ := args["account"].(string)
	minScore := 70
	if score, ok := args["min_score"].(float64); ok {
		minScore = int(score)
	}
	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	// Get priority emails from database
	emails, err := ies.priority.GetPriorityEmails(accountID, minScore, limit)
	if err != nil {
		return "", fmt.Errorf("failed to get priority emails: %w", err)
	}

	if len(emails) == 0 {
		return fmt.Sprintf("✅ No emails found with priority score >= %d", minScore), nil
	}

	// Format response
	response := fmt.Sprintf("🎯 Priority Inbox (score >= %d)\n\n", minScore)
	response += fmt.Sprintf("Found %d high-priority emails:\n\n", len(emails))

	for i, email := range emails {
		// Get priority details
		priority, _ := ies.db.GetPriority(email.ID)
		classification, _ := ies.db.GetClassification(email.ID)

		priorityIcon := getPriorityIcon(priority.Score)
		categoryLabel := ""
		if classification != nil {
			categoryLabel = fmt.Sprintf("[%s] ", classification.Category)
		}

		response += fmt.Sprintf("%d. %s %s%s\n", i+1, priorityIcon, categoryLabel, email.Subject)
		response += fmt.Sprintf("   From: %s\n", email.From)
		response += fmt.Sprintf("   Score: %d/100\n", priority.Score)
		response += fmt.Sprintf("   Received: %s\n", email.ReceivedAt.Format("2006-01-02 15:04"))
		if priority.Reasoning != "" {
			response += fmt.Sprintf("   Why: %s\n", truncate(priority.Reasoning, 100))
		}
		response += "\n"
	}

	return response, nil
}

// HandleSmartFilter handles the smart_filter tool
func (ies *IntelligentEmailServer) HandleSmartFilter(args map[string]interface{}) (string, error) {
	accountID, _ := args["account"].(string)
	category, _ := args["category"].(string)
	minPriority := 0
	if mp, ok := args["min_priority"].(float64); ok {
		minPriority = int(mp)
	}
	limit := 50
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	// Build filter
	filter := storage.EmailFilter{
		AccountID:   accountID,
		Category:    category,
		MinPriority: minPriority,
		Limit:       limit,
	}

	// Handle unread_only
	if unreadOnly, ok := args["unread_only"].(bool); ok && unreadOnly {
		filter.Read = &unreadOnly
	}

	// Handle date filters
	if dateFrom, ok := args["date_from"].(string); ok && dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			filter.DateFrom = t
		} else if t, err := time.Parse(time.RFC3339, dateFrom); err == nil {
			filter.DateFrom = t
		}
	}
	if dateTo, ok := args["date_to"].(string); ok && dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			filter.DateTo = t
		} else if t, err := time.Parse(time.RFC3339, dateTo); err == nil {
			filter.DateTo = t
		}
	}

	// Query database
	emails, err := ies.db.ListEmails(filter)
	if err != nil {
		return "", fmt.Errorf("failed to filter emails: %w", err)
	}

	if len(emails) == 0 {
		return "No emails found matching the criteria", nil
	}

	// Format response
	response := "🔍 Smart Filter Results\n\n"
	response += fmt.Sprintf("Filters applied:\n")
	if accountID != "" {
		response += fmt.Sprintf("  • Account: %s\n", accountID)
	}
	if category != "" {
		response += fmt.Sprintf("  • Category: %s\n", category)
	}
	if minPriority > 0 {
		response += fmt.Sprintf("  • Min Priority: %d\n", minPriority)
	}
	response += fmt.Sprintf("\nFound %d emails:\n\n", len(emails))

	for i, email := range emails {
		classification, _ := ies.db.GetClassification(email.ID)
		priority, _ := ies.db.GetPriority(email.ID)

		categoryLabel := "unknown"
		priorityScore := 0
		if classification != nil {
			categoryLabel = classification.Category
		}
		if priority != nil {
			priorityScore = priority.Score
		}

		response += fmt.Sprintf("%d. [%s] %s\n", i+1, categoryLabel, email.Subject)
		response += fmt.Sprintf("   From: %s | Priority: %d/100\n", email.From, priorityScore)
		response += fmt.Sprintf("   Date: %s\n\n", email.ReceivedAt.Format("2006-01-02 15:04"))
	}

	return response, nil
}

// HandleAnalyzePriority handles the analyze_priority tool
func (ies *IntelligentEmailServer) HandleAnalyzePriority(args map[string]interface{}) (string, error) {
	emailID, _ := args["email_id"].(string)
	from, _ := args["from"].(string)
	subject, _ := args["subject"].(string)
	bodySnippet, _ := args["body_snippet"].(string)
	receivedAtStr, _ := args["received_at"].(string)

	// If email_id provided, get from database
	if emailID != "" {
		priority, err := ies.priority.GetPriorityBreakdown(emailID)
		if err != nil {
			return "", fmt.Errorf("failed to get priority breakdown: %w", err)
		}

		explanation := ies.priority.ExplainPriority(priority)
		return explanation, nil
	}

	// Otherwise, analyze on-the-fly
	if from == "" || subject == "" {
		return "", fmt.Errorf("from and subject are required")
	}

	receivedAt := time.Now()
	if receivedAtStr != "" {
		if t, err := time.Parse(time.RFC3339, receivedAtStr); err == nil {
			receivedAt = t
		}
	}

	email := &ai.Email{
		From:        from,
		Subject:     subject,
		BodySnippet: bodySnippet,
		ReceivedAt:  receivedAt,
	}

	priority, err := ies.priority.CalculatePriority(email)
	if err != nil {
		return "", fmt.Errorf("failed to calculate priority: %w", err)
	}

	explanation := ies.priority.ExplainPriority(priority)
	return explanation, nil
}

// Helper functions

func getPriorityIcon(score int) string {
	if score >= 90 {
		return "🔴"
	} else if score >= 70 {
		return "🟠"
	} else if score >= 40 {
		return "🟡"
	} else if score >= 20 {
		return "🟢"
	}
	return "⚪"
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// GetStats returns statistics about the intelligent email system
func (ies *IntelligentEmailServer) GetStats() (map[string]interface{}, error) {
	dbStats, err := ies.db.Stats()
	if err != nil {
		return nil, err
	}

	classifierStats := ies.classifier.GetStats()
	priorityStats := ies.priority.GetStats()

	stats := map[string]interface{}{
		"database":   dbStats,
		"classifier": classifierStats,
		"priority":   priorityStats,
	}

	statsJSON, _ := json.MarshalIndent(stats, "", "  ")
	return map[string]interface{}{
		"stats": string(statsJSON),
	}, nil
}

// Close closes the intelligent email server
func (ies *IntelligentEmailServer) Close() error {
	if ies.db != nil {
		return ies.db.Close()
	}
	return nil
}
