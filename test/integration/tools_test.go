package integration

import (
	"email-mcp-server/ai"
	"email-mcp-server/server"
	"email-mcp-server/storage"
	"email-mcp-server/utils"
	"fmt"
	"os"
	"testing"
	"time"
)

// setupTestServer creates a test IntelligentEmailServer with temporary database
func setupTestServer(t *testing.T) (*server.IntelligentEmailServer, func()) {
	// Create temporary database
	tmpDB := fmt.Sprintf("/tmp/test_emails_%d.db", time.Now().UnixNano())

	// Initialize server
	intelligentServer, err := server.NewIntelligentEmailServer(
		tmpDB,
		"../../config/priority_rules.example.json",
	)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	// Cleanup function
	cleanup := func() {
		intelligentServer.Close()
		os.Remove(tmpDB)
	}

	return intelligentServer, cleanup
}

// seedTestEmails adds test emails to the database
func seedTestEmails(t *testing.T, intelligentServer *server.IntelligentEmailServer) []string {
	// Access the internal database (this requires the server to expose it or we use reflection)
	// For now, we'll create emails through the classification tool

	testEmails := []struct {
		id          string
		from        string
		subject     string
		bodySnippet string
	}{
		{
			id:          "test-email-1",
			from:        "boss@company.com",
			subject:     "URGENT: Q4 Budget Review",
			bodySnippet: "Please review the budget proposal immediately",
		},
		{
			id:          "test-email-2",
			from:        "newsletter@techcrunch.com",
			subject:     "Daily Tech Digest",
			bodySnippet: "Here are today's top stories...",
		},
		{
			id:          "test-email-3",
			from:        "billing@vendor.com",
			subject:     "Invoice #12345 - Payment Due",
			bodySnippet: "Your invoice is attached",
		},
		{
			id:          "test-email-4",
			from:        "colleague@company.com",
			subject:     "Re: Project Status",
			bodySnippet: "Thanks for the update",
		},
		{
			id:          "test-email-5",
			from:        "sales@store.com",
			subject:     "50% OFF - Limited Time",
			bodySnippet: "Don't miss our biggest sale",
		},
	}

	var emailIDs []string
	for _, te := range testEmails {
		args := map[string]interface{}{
			"email_id":     te.id,
			"from":         te.from,
			"subject":      te.subject,
			"body_snippet": te.bodySnippet,
		}

		_, err := intelligentServer.HandleClassifyEmail(args)
		if err != nil {
			t.Logf("Warning: Failed to classify test email %s: %v", te.id, err)
		}

		emailIDs = append(emailIDs, te.id)
	}

	return emailIDs
}

func TestTools_ClassifyEmail(t *testing.T) {
	intelligentServer, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name             string
		args             map[string]interface{}
		expectError      bool
		expectedCategory string
	}{
		{
			name: "VIP urgent email",
			args: map[string]interface{}{
				"from":         "boss@company.com",
				"subject":      "URGENT: Critical Issue",
				"body_snippet": "Immediate action required",
			},
			expectError:      false,
			expectedCategory: "urgent",
		},
		{
			name: "Newsletter email",
			args: map[string]interface{}{
				"from":         "newsletter@example.com",
				"subject":      "Weekly Tech News",
				"body_snippet": "Unsubscribe at any time",
			},
			expectError:      false,
			expectedCategory: "newsletters",
		},
		{
			name: "Invoice email",
			args: map[string]interface{}{
				"from":         "billing@vendor.com",
				"subject":      "Invoice #999",
				"body_snippet": "Payment due next week",
			},
			expectError:      false,
			expectedCategory: "invoice",
		},
		{
			name: "Promotional email",
			args: map[string]interface{}{
				"from":         "sales@store.com",
				"subject":      "50% OFF Sale",
				"body_snippet": "Limited time offer",
			},
			expectError:      false,
			expectedCategory: "promotions",
		},
		{
			name: "Missing required fields",
			args: map[string]interface{}{
				"subject": "Test",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := intelligentServer.HandleClassifyEmail(tt.args)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("HandleClassifyEmail() error = %v", err)
			}

			if len(result) == 0 {
				t.Error("HandleClassifyEmail() returned empty result")
			}

			t.Logf("✅ %s classified successfully", tt.name)
			t.Logf("   Result:\n%s", result)
		})
	}
}

func TestTools_PriorityInbox(t *testing.T) {
	intelligentServer, cleanup := setupTestServer(t)
	defer cleanup()

	// Seed test emails
	seedTestEmails(t, intelligentServer)

	tests := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
	}{
		{
			name: "Get high priority emails (score >= 70)",
			args: map[string]interface{}{
				"min_score": float64(70),
				"limit":     float64(10),
			},
			expectError: false,
		},
		{
			name: "Get all priority emails (score >= 0)",
			args: map[string]interface{}{
				"min_score": float64(0),
				"limit":     float64(20),
			},
			expectError: false,
		},
		{
			name: "Get critical priority (score >= 90)",
			args: map[string]interface{}{
				"min_score": float64(90),
				"limit":     float64(5),
			},
			expectError: false,
		},
		{
			name:        "Default parameters",
			args:        map[string]interface{}{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := intelligentServer.HandlePriorityInbox(tt.args)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("HandlePriorityInbox() error = %v", err)
			}

			if len(result) == 0 {
				t.Error("HandlePriorityInbox() returned empty result")
			}

			t.Logf("✅ %s completed", tt.name)
			t.Logf("   Result:\n%s", result)
		})
	}
}

func TestTools_SmartFilter(t *testing.T) {
	intelligentServer, cleanup := setupTestServer(t)
	defer cleanup()

	// Seed test emails
	seedTestEmails(t, intelligentServer)

	tests := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
	}{
		{
			name: "Filter by work category",
			args: map[string]interface{}{
				"category": "work",
				"limit":    float64(10),
			},
			expectError: false,
		},
		{
			name: "Filter by newsletters",
			args: map[string]interface{}{
				"category": "newsletters",
				"limit":    float64(20),
			},
			expectError: false,
		},
		{
			name: "Filter by invoice category",
			args: map[string]interface{}{
				"category": "invoice",
				"limit":    float64(10),
			},
			expectError: false,
		},
		{
			name: "Filter by priority score",
			args: map[string]interface{}{
				"min_priority": float64(50),
				"limit":        float64(15),
			},
			expectError: false,
		},
		{
			name: "Filter unread emails",
			args: map[string]interface{}{
				"unread_only": true,
				"limit":       float64(25),
			},
			expectError: false,
		},
		{
			name: "Filter by date range",
			args: map[string]interface{}{
				"date_from": time.Now().Add(-7 * 24 * time.Hour).Format("2006-01-02"),
				"date_to":   time.Now().Format("2006-01-02"),
				"limit":     float64(50),
			},
			expectError: false,
		},
		{
			name: "Combined filters",
			args: map[string]interface{}{
				"category":     "work",
				"min_priority": float64(60),
				"unread_only":  true,
				"limit":        float64(10),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := intelligentServer.HandleSmartFilter(tt.args)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("HandleSmartFilter() error = %v", err)
			}

			if len(result) == 0 {
				t.Error("HandleSmartFilter() returned empty result")
			}

			t.Logf("✅ %s completed", tt.name)
			t.Logf("   Result:\n%s", result)
		})
	}
}

func TestTools_AnalyzePriority(t *testing.T) {
	intelligentServer, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
	}{
		{
			name: "Analyze VIP urgent email",
			args: map[string]interface{}{
				"from":         "boss@company.com",
				"subject":      "URGENT: Critical Budget Issue",
				"body_snippet": "Action required immediately",
				"received_at":  time.Now().Format(time.RFC3339),
			},
			expectError: false,
		},
		{
			name: "Analyze newsletter",
			args: map[string]interface{}{
				"from":         "newsletter@example.com",
				"subject":      "Weekly Digest",
				"body_snippet": "This week's top stories",
				"received_at":  time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			},
			expectError: false,
		},
		{
			name: "Analyze old promotional email",
			args: map[string]interface{}{
				"from":         "sales@store.com",
				"subject":      "Sale ending soon",
				"body_snippet": "Last chance to save",
				"received_at":  time.Now().Add(-5 * 24 * time.Hour).Format(time.RFC3339),
			},
			expectError: false,
		},
		{
			name: "Missing required fields",
			args: map[string]interface{}{
				"subject": "Test",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := intelligentServer.HandleAnalyzePriority(tt.args)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("HandleAnalyzePriority() error = %v", err)
			}

			if len(result) == 0 {
				t.Error("HandleAnalyzePriority() returned empty result")
			}

			t.Logf("✅ %s completed", tt.name)
			t.Logf("   Result:\n%s", result)
		})
	}
}

func TestTools_Integration_FullWorkflow(t *testing.T) {
	intelligentServer, cleanup := setupTestServer(t)
	defer cleanup()

	// Step 1: Classify an email
	t.Log("Step 1: Classifying email...")
	classifyArgs := map[string]interface{}{
		"email_id":     "workflow-test-1",
		"from":         "boss@company.com",
		"subject":      "URGENT: Q4 Planning Meeting",
		"body_snippet": "Please review before tomorrow's meeting",
	}

	classifyResult, err := intelligentServer.HandleClassifyEmail(classifyArgs)
	if err != nil {
		t.Fatalf("Classification failed: %v", err)
	}
	t.Logf("✅ Classification result:\n%s\n", classifyResult)

	// Step 2: Analyze priority
	t.Log("Step 2: Analyzing priority...")
	analyzeArgs := map[string]interface{}{
		"from":         "boss@company.com",
		"subject":      "URGENT: Q4 Planning Meeting",
		"body_snippet": "Please review before tomorrow's meeting",
		"received_at":  time.Now().Format(time.RFC3339),
	}

	analyzeResult, err := intelligentServer.HandleAnalyzePriority(analyzeArgs)
	if err != nil {
		t.Fatalf("Priority analysis failed: %v", err)
	}
	t.Logf("✅ Priority analysis:\n%s\n", analyzeResult)

	// Step 3: Add more test emails
	t.Log("Step 3: Adding more test emails...")
	seedTestEmails(t, intelligentServer)

	// Step 4: Get priority inbox
	t.Log("Step 4: Fetching priority inbox...")
	priorityArgs := map[string]interface{}{
		"min_score": float64(70),
		"limit":     float64(10),
	}

	priorityResult, err := intelligentServer.HandlePriorityInbox(priorityArgs)
	if err != nil {
		t.Fatalf("Priority inbox failed: %v", err)
	}
	t.Logf("✅ Priority inbox:\n%s\n", priorityResult)

	// Step 5: Apply smart filters
	t.Log("Step 5: Applying smart filters...")
	filterArgs := map[string]interface{}{
		"category": "work",
		"limit":    float64(20),
	}

	filterResult, err := intelligentServer.HandleSmartFilter(filterArgs)
	if err != nil {
		t.Fatalf("Smart filter failed: %v", err)
	}
	t.Logf("✅ Smart filter:\n%s\n", filterResult)

	t.Log("\n✅ Full workflow integration test passed!")
}

func TestTools_Integration_EmailConversion(t *testing.T) {
	// Test the utility functions for email conversion
	storageEmail := &storage.Email{
		ID:          "test-conversion",
		AccountID:   "account1",
		From:        "test@example.com",
		To:          "user@example.com",
		Subject:     "Test Email",
		BodySnippet: "This is a test",
		ReceivedAt:  time.Now(),
		Read:        false,
		Starred:     false,
		Deleted:     false,
	}

	aiEmail := utils.EmailToAIEmail(storageEmail)

	if aiEmail.ID != storageEmail.ID {
		t.Errorf("ID mismatch: got %s, want %s", aiEmail.ID, storageEmail.ID)
	}
	if aiEmail.From != storageEmail.From {
		t.Errorf("From mismatch: got %s, want %s", aiEmail.From, storageEmail.From)
	}
	if aiEmail.Subject != storageEmail.Subject {
		t.Errorf("Subject mismatch: got %s, want %s", aiEmail.Subject, storageEmail.Subject)
	}

	t.Log("✅ Email conversion test passed")
}

func TestTools_Integration_GenerateEmailID(t *testing.T) {
	accountID := "account1"
	messageID := "msg123"
	receivedAt := time.Now()

	// Test with message ID
	id1 := utils.GenerateEmailID(accountID, messageID, receivedAt)
	expectedID1 := fmt.Sprintf("%s:%s", accountID, messageID)
	if id1 != expectedID1 {
		t.Errorf("GenerateEmailID() with messageID = %s, want %s", id1, expectedID1)
	}

	// Test without message ID
	id2 := utils.GenerateEmailID(accountID, "", receivedAt)
	expectedID2 := fmt.Sprintf("%s:%d", accountID, receivedAt.Unix())
	if id2 != expectedID2 {
		t.Errorf("GenerateEmailID() without messageID = %s, want %s", id2, expectedID2)
	}

	t.Log("✅ Email ID generation test passed")
}

func TestTools_Integration_BatchClassification(t *testing.T) {
	intelligentServer, cleanup := setupTestServer(t)
	defer cleanup()

	// Classify multiple emails in sequence
	emails := []struct {
		from    string
		subject string
		body    string
	}{
		{"boss@company.com", "URGENT: Meeting", "Please attend"},
		{"newsletter@tech.com", "Tech Weekly", "Unsubscribe link"},
		{"billing@vendor.com", "Invoice #123", "Payment due"},
		{"sales@store.com", "Big Sale!", "50% off everything"},
		{"colleague@company.com", "Project Update", "Status report"},
	}

	for i, email := range emails {
		args := map[string]interface{}{
			"email_id":     fmt.Sprintf("batch-test-%d", i),
			"from":         email.from,
			"subject":      email.subject,
			"body_snippet": email.body,
		}

		result, err := intelligentServer.HandleClassifyEmail(args)
		if err != nil {
			t.Errorf("Batch classification failed for email %d: %v", i, err)
			continue
		}

		t.Logf("✅ Email %d classified: %s", i+1, email.subject)
		t.Logf("   From: %s", email.from)
	}

	t.Logf("\n✅ Batch classification of %d emails completed", len(emails))
}

func TestTools_Integration_ErrorHandling(t *testing.T) {
	intelligentServer, cleanup := setupTestServer(t)
	defer cleanup()

	// Test various error conditions
	tests := []struct {
		name    string
		handler func(map[string]interface{}) (string, error)
		args    map[string]interface{}
	}{
		{
			name:    "Classify with missing from",
			handler: intelligentServer.HandleClassifyEmail,
			args: map[string]interface{}{
				"subject": "Test",
			},
		},
		{
			name:    "Classify with missing subject",
			handler: intelligentServer.HandleClassifyEmail,
			args: map[string]interface{}{
				"from": "test@example.com",
			},
		},
		{
			name:    "Analyze with missing from",
			handler: intelligentServer.HandleAnalyzePriority,
			args: map[string]interface{}{
				"subject": "Test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.handler(tt.args)
			if err == nil {
				t.Error("Expected error but got none")
			} else {
				t.Logf("✅ Error correctly caught: %v", err)
			}
		})
	}
}

func TestTools_Integration_Stats(t *testing.T) {
	intelligentServer, cleanup := setupTestServer(t)
	defer cleanup()

	// Seed test emails
	seedTestEmails(t, intelligentServer)

	// Get stats
	stats, err := intelligentServer.GetStats()
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if len(stats) == 0 {
		t.Error("GetStats() returned empty stats")
	}

	t.Logf("✅ Stats retrieved successfully")
	for key, value := range stats {
		t.Logf("   %s: %v", key, value)
	}
}

func TestTools_Integration_ConcurrentAccess(t *testing.T) {
	intelligentServer, cleanup := setupTestServer(t)
	defer cleanup()

	// Test concurrent tool calls
	done := make(chan bool)

	// Launch multiple concurrent classifications
	for i := 0; i < 5; i++ {
		go func(index int) {
			args := map[string]interface{}{
				"from":         fmt.Sprintf("user%d@example.com", index),
				"subject":      fmt.Sprintf("Test Email %d", index),
				"body_snippet": fmt.Sprintf("Test content %d", index),
			}

			_, err := intelligentServer.HandleClassifyEmail(args)
			if err != nil {
				t.Errorf("Concurrent classification %d failed: %v", index, err)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	t.Log("✅ Concurrent access test passed")
}
