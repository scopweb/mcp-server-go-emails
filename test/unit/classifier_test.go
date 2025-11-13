package unit

import (
	"email-mcp-server/ai"
	"email-mcp-server/config"
	"testing"
	"time"
)

func TestClassifier_Classify(t *testing.T) {
	// Load test config
	cfg, err := config.LoadPriorityConfig("../../config/priority_rules.example.json")
	if err != nil {
		t.Skipf("Skipping test - config file not found: %v", err)
		return
	}

	classifier := ai.NewClassifier(cfg, nil) // nil DB for unit tests

	tests := []struct {
		name             string
		email            *ai.Email
		expectedCategory string
		minConfidence    float64
	}{
		{
			name: "Newsletter email",
			email: &ai.Email{
				ID:          "test-1",
				From:        "newsletter@techcrunch.com",
				Subject:     "Daily Tech News Digest - January 12",
				BodySnippet: "Here are today's top tech stories...",
				ReceivedAt:  time.Now(),
			},
			expectedCategory: "newsletters",
			minConfidence:    0.7,
		},
		{
			name: "Promotional email",
			email: &ai.Email{
				ID:          "test-2",
				From:        "sales@store.com",
				Subject:     "50% OFF SALE - Limited Time Only!",
				BodySnippet: "Don't miss out on our biggest sale...",
				ReceivedAt:  time.Now(),
			},
			expectedCategory: "promotions",
			minConfidence:    0.7,
		},
		{
			name: "Invoice email",
			email: &ai.Email{
				ID:          "test-3",
				From:        "billing@vendor.com",
				Subject:     "Invoice #12345 - Payment Due",
				BodySnippet: "Your invoice is attached...",
				ReceivedAt:  time.Now(),
			},
			expectedCategory: "invoice",
			minConfidence:    0.8,
		},
		{
			name: "Urgent email",
			email: &ai.Email{
				ID:          "test-4",
				From:        "security@company.com",
				Subject:     "URGENT: Security Alert",
				BodySnippet: "Immediate action required...",
				ReceivedAt:  time.Now(),
			},
			expectedCategory: "urgent",
			minConfidence:    0.85,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := classifier.Classify(tt.email)
			if err != nil {
				t.Fatalf("Classify() error = %v", err)
			}

			if result.Category != tt.expectedCategory {
				t.Errorf("Classify() category = %v, want %v", result.Category, tt.expectedCategory)
			}

			if result.Confidence < tt.minConfidence {
				t.Errorf("Classify() confidence = %v, want >= %v", result.Confidence, tt.minConfidence)
			}

			t.Logf("✅ %s classified as '%s' (confidence: %.2f, method: %s)",
				tt.name, result.Category, result.Confidence, result.Method)
			t.Logf("   Reasoning: %s", result.Reasoning)
		})
	}
}

func TestClassifier_ClassifyBatch(t *testing.T) {
	cfg, err := config.LoadPriorityConfig("../../config/priority_rules.example.json")
	if err != nil {
		t.Skipf("Skipping test - config file not found: %v", err)
		return
	}

	classifier := ai.NewClassifier(cfg, nil)

	emails := []*ai.Email{
		{
			ID:          "batch-1",
			From:        "newsletter@example.com",
			Subject:     "Newsletter: Top Stories",
			BodySnippet: "Here are this week's highlights...",
			ReceivedAt:  time.Now(),
		},
		{
			ID:          "batch-2",
			From:        "invoice@vendor.com",
			Subject:     "Invoice #999",
			BodySnippet: "Payment due next week...",
			ReceivedAt:  time.Now(),
		},
		{
			ID:          "batch-3",
			From:        "urgent@example.com",
			Subject:     "CRITICAL: Server Down",
			BodySnippet: "Immediate attention needed...",
			ReceivedAt:  time.Now(),
		},
	}

	results, err := classifier.ClassifyBatch(emails)
	if err != nil {
		t.Fatalf("ClassifyBatch() error = %v", err)
	}

	if len(results) != len(emails) {
		t.Errorf("ClassifyBatch() returned %d results, want %d", len(results), len(emails))
	}

	t.Logf("✅ Batch classification completed: %d emails processed", len(results))
}

func TestClassifier_Cache(t *testing.T) {
	cfg, err := config.LoadPriorityConfig("../../config/priority_rules.example.json")
	if err != nil {
		t.Skipf("Skipping test - config file not found: %v", err)
		return
	}

	classifier := ai.NewClassifier(cfg, nil)

	email := &ai.Email{
		ID:          "cache-test",
		From:        "test@example.com",
		Subject:     "Test Email",
		BodySnippet: "This is a test",
		ReceivedAt:  time.Now(),
	}

	// First classification
	result1, err := classifier.Classify(email)
	if err != nil {
		t.Fatalf("First Classify() error = %v", err)
	}

	// Second classification (should use cache)
	result2, err := classifier.Classify(email)
	if err != nil {
		t.Fatalf("Second Classify() error = %v", err)
	}

	if result1.Category != result2.Category {
		t.Errorf("Cached result different: got %s, want %s", result2.Category, result1.Category)
	}

	t.Logf("✅ Cache working correctly")
}
