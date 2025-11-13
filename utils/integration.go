package utils

import (
	"email-mcp-server/ai"
	"email-mcp-server/server"
	"email-mcp-server/storage"
	"fmt"
	"time"
)

// EmailToAIEmail converts storage.Email to ai.Email
func EmailToAIEmail(email *storage.Email) *ai.Email {
	return &ai.Email{
		ID:          email.ID,
		From:        email.From,
		To:          email.To,
		Subject:     email.Subject,
		BodySnippet: email.BodySnippet,
		ReceivedAt:  email.ReceivedAt,
	}
}

// GenerateEmailID generates a unique email ID
func GenerateEmailID(accountID string, messageID string, receivedAt time.Time) string {
	if messageID != "" {
		return fmt.Sprintf("%s:%s", accountID, messageID)
	}
	return fmt.Sprintf("%s:%d", accountID, receivedAt.Unix())
}

// IntegrationExample demonstrates how to use the intelligent email server
func IntegrationExample() {
	// Example: Initialize intelligent email server
	intelligentServer, err := server.NewIntelligentEmailServer(
		"./data/emails.db",                   // Database path
		"./config/priority_rules.example.json", // Config path
	)
	if err != nil {
		fmt.Printf("Error initializing server: %v\n", err)
		return
	}
	defer intelligentServer.Close()

	// Example: Classify an email
	classificationArgs := map[string]interface{}{
		"from":         "boss@company.com",
		"subject":      "URGENT: Q4 Budget Review Meeting Tomorrow",
		"body_snippet": "Please review the attached budget proposal before tomorrow's meeting...",
	}

	result, err := intelligentServer.HandleClassifyEmail(classificationArgs)
	if err != nil {
		fmt.Printf("Classification error: %v\n", err)
	} else {
		fmt.Println(result)
	}

	// Example: Get priority inbox
	priorityArgs := map[string]interface{}{
		"min_score": float64(70),
		"limit":     float64(10),
	}

	priorityResult, err := intelligentServer.HandlePriorityInbox(priorityArgs)
	if err != nil {
		fmt.Printf("Priority inbox error: %v\n", err)
	} else {
		fmt.Println(priorityResult)
	}

	// Example: Smart filter
	filterArgs := map[string]interface{}{
		"category":    "work",
		"unread_only": true,
		"limit":       float64(20),
	}

	filterResult, err := intelligentServer.HandleSmartFilter(filterArgs)
	if err != nil {
		fmt.Printf("Smart filter error: %v\n", err)
	} else {
		fmt.Println(filterResult)
	}
}

// MCPToolCallExample shows how to integrate with MCP protocol
func MCPToolCallExample(toolName string, arguments map[string]interface{}, intelligentServer *server.IntelligentEmailServer) (string, error) {
	switch toolName {
	case "classify_email":
		return intelligentServer.HandleClassifyEmail(arguments)

	case "priority_inbox":
		return intelligentServer.HandlePriorityInbox(arguments)

	case "smart_filter":
		return intelligentServer.HandleSmartFilter(arguments)

	case "analyze_priority":
		return intelligentServer.HandleAnalyzePriority(arguments)

	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

// SyncEmailsToDatabase syncs IMAP emails to local database for intelligent processing
func SyncEmailsToDatabase(
	imapEmails []interface{}, // Your IMAP emails
	accountID string,
	db *storage.Database,
	classifier *ai.Classifier,
	priorityEngine *ai.PriorityEngine,
) error {
	for _, imapEmail := range imapEmails {
		// Convert IMAP email to our format (you'll need to implement this based on your IMAP structure)
		// This is a placeholder showing the pattern
		email := convertIMAPToStorage(imapEmail, accountID)

		// Save to database
		if err := db.CreateEmail(email); err != nil {
			return fmt.Errorf("failed to save email: %w", err)
		}

		// Classify the email
		aiEmail := EmailToAIEmail(email)
		classification, err := classifier.Classify(aiEmail)
		if err != nil {
			return fmt.Errorf("failed to classify email: %w", err)
		}

		// Save classification
		if err := classifier.SaveClassification(classification); err != nil {
			return fmt.Errorf("failed to save classification: %w", err)
		}

		// Calculate priority
		priority, err := priorityEngine.CalculatePriority(aiEmail)
		if err != nil {
			return fmt.Errorf("failed to calculate priority: %w", err)
		}

		// Save priority
		if err := priorityEngine.SavePriority(priority); err != nil {
			return fmt.Errorf("failed to save priority: %w", err)
		}
	}

	return nil
}

// convertIMAPToStorage is a placeholder - implement based on your IMAP email structure
func convertIMAPToStorage(imapEmail interface{}, accountID string) *storage.Email {
	// TODO: Implement conversion from your IMAP email type to storage.Email
	// This is just an example structure
	return &storage.Email{
		ID:          GenerateEmailID(accountID, "", time.Now()),
		AccountID:   accountID,
		From:        "example@example.com",
		To:          "user@example.com",
		Subject:     "Example",
		BodySnippet: "Example body",
		ReceivedAt:  time.Now(),
		Read:        false,
		Starred:     false,
		Deleted:     false,
	}
}
