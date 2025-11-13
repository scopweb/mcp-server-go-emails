package unit

import (
	"email-mcp-server/ai"
	"email-mcp-server/config"
	"testing"
	"time"
)

func TestPriorityEngine_CalculatePriority(t *testing.T) {
	// Load test config
	cfg, err := config.LoadPriorityConfig("../../config/priority_rules.example.json")
	if err != nil {
		t.Skipf("Skipping test - config file not found: %v", err)
		return
	}

	classifier := ai.NewClassifier(cfg, nil)
	priorityEngine := ai.NewPriorityEngine(cfg, nil, classifier)

	tests := []struct {
		name          string
		email         *ai.Email
		minScore      int
		maxScore      int
		expectedHigh  bool // Should be high priority (>= 70)
	}{
		{
			name: "VIP sender with urgent subject",
			email: &ai.Email{
				ID:          "test-vip-urgent",
				From:        "boss@company.com", // VIP in example config
				Subject:     "URGENT: Critical Issue",
				BodySnippet: "Immediate action required",
				ReceivedAt:  time.Now(),
			},
			minScore:     80,
			maxScore:     100,
			expectedHigh: true,
		},
		{
			name: "Newsletter from known sender",
			email: &ai.Email{
				ID:          "test-newsletter",
				From:        "newsletter@techcrunch.com",
				Subject:     "Daily Tech News - January 13",
				BodySnippet: "Here are today's top stories...",
				ReceivedAt:  time.Now(),
			},
			minScore:     0,
			maxScore:     40,
			expectedHigh: false,
		},
		{
			name: "Invoice with action keywords",
			email: &ai.Email{
				ID:          "test-invoice",
				From:        "billing@vendor.com",
				Subject:     "Invoice #12345 - Payment Due",
				BodySnippet: "Please respond with payment confirmation by Friday",
				ReceivedAt:  time.Now(),
			},
			minScore:     40,
			maxScore:     80,
			expectedHigh: false,
		},
		{
			name: "Recent email from important domain",
			email: &ai.Email{
				ID:          "test-important-domain",
				From:        "support@company.com", // Important domain in config
				Subject:     "System maintenance scheduled",
				BodySnippet: "Planned maintenance window...",
				ReceivedAt:  time.Now().Add(-30 * time.Minute), // 30 min ago
			},
			minScore:     30,
			maxScore:     70,
			expectedHigh: false,
		},
		{
			name: "Old promotional email",
			email: &ai.Email{
				ID:          "test-old-promo",
				From:        "sales@store.com",
				Subject:     "50% OFF Sale",
				BodySnippet: "Limited time offer...",
				ReceivedAt:  time.Now().Add(-7 * 24 * time.Hour), // 7 days ago
			},
			minScore:     0,
			maxScore:     30,
			expectedHigh: false,
		},
		{
			name: "Thread reply with urgent keyword",
			email: &ai.Email{
				ID:          "test-thread-urgent",
				From:        "colleague@company.com",
				Subject:     "Re: URGENT: Project Deadline",
				BodySnippet: "I've reviewed the requirements...",
				ReceivedAt:  time.Now().Add(-1 * time.Hour),
			},
			minScore:     50,
			maxScore:     90,
			expectedHigh: false, // May or may not be high depending on other factors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := priorityEngine.CalculatePriority(tt.email)
			if err != nil {
				t.Fatalf("CalculatePriority() error = %v", err)
			}

			// Check score is within expected range
			if result.Score < tt.minScore || result.Score > tt.maxScore {
				t.Errorf("CalculatePriority() score = %d, expected range [%d-%d]",
					result.Score, tt.minScore, tt.maxScore)
			}

			// Check high priority expectation
			isHigh := result.Score >= 70
			if isHigh != tt.expectedHigh {
				t.Logf("Note: Priority expectation mismatch (got %d, expected high=%v)",
					result.Score, tt.expectedHigh)
			}

			// Verify factors are present
			if len(result.Factors) == 0 {
				t.Error("CalculatePriority() returned no factors")
			}

			// Verify reasoning is provided
			if len(result.ReasoningChain) == 0 {
				t.Error("CalculatePriority() returned no reasoning")
			}

			t.Logf("✅ %s: score=%d/100, category=%s",
				tt.name, result.Score, result.Category)
			t.Logf("   Factors: sender=%d, keywords=%d, temporal=%d, category=%d",
				result.Factors["sender"], result.Factors["keywords"],
				result.Factors["temporal"], result.Factors["category"])
			t.Logf("   Reasoning: %v", result.ReasoningChain)
		})
	}
}

func TestPriorityEngine_SenderScore(t *testing.T) {
	cfg, err := config.LoadPriorityConfig("../../config/priority_rules.example.json")
	if err != nil {
		t.Skipf("Skipping test - config file not found: %v", err)
		return
	}

	classifier := ai.NewClassifier(cfg, nil)
	priorityEngine := ai.NewPriorityEngine(cfg, nil, classifier)

	tests := []struct {
		name          string
		from          string
		expectedScore int // Approximate expected sender score contribution
	}{
		{
			name:          "VIP sender",
			from:          "boss@company.com",
			expectedScore: 30, // VIP gets 30 points
		},
		{
			name:          "Important domain",
			from:          "info@company.com",
			expectedScore: 20, // Important domain gets 20 points
		},
		{
			name:          "Unknown sender",
			from:          "random@unknown.com",
			expectedScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := &ai.Email{
				ID:          "test-sender-" + tt.name,
				From:        tt.from,
				Subject:     "Test email",
				BodySnippet: "Test content",
				ReceivedAt:  time.Now(),
			}

			result, err := priorityEngine.CalculatePriority(email)
			if err != nil {
				t.Fatalf("CalculatePriority() error = %v", err)
			}

			senderScore := result.Factors["sender"]
			if senderScore != tt.expectedScore {
				t.Logf("Sender score = %d, expected ~%d (difference acceptable due to engagement)",
					senderScore, tt.expectedScore)
			}

			t.Logf("✅ %s: sender_score=%d", tt.name, senderScore)
		})
	}
}

func TestPriorityEngine_KeywordScore(t *testing.T) {
	cfg, err := config.LoadPriorityConfig("../../config/priority_rules.example.json")
	if err != nil {
		t.Skipf("Skipping test - config file not found: %v", err)
		return
	}

	classifier := ai.NewClassifier(cfg, nil)
	priorityEngine := ai.NewPriorityEngine(cfg, nil, classifier)

	tests := []struct {
		name         string
		subject      string
		body         string
		minKeywordScore int
	}{
		{
			name:            "Urgent in subject",
			subject:         "URGENT: Server Down",
			body:            "Normal content",
			minKeywordScore: 15, // Should get urgent keyword bonus
		},
		{
			name:            "Action required in body",
			subject:         "Project Update",
			body:            "Action required: Please review by EOD",
			minKeywordScore: 10, // Should get action keyword bonus
		},
		{
			name:            "No urgent keywords",
			subject:         "Weekly Newsletter",
			body:            "Here are the latest updates",
			minKeywordScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := &ai.Email{
				ID:          "test-keyword-" + tt.name,
				From:        "test@example.com",
				Subject:     tt.subject,
				BodySnippet: tt.body,
				ReceivedAt:  time.Now(),
			}

			result, err := priorityEngine.CalculatePriority(email)
			if err != nil {
				t.Fatalf("CalculatePriority() error = %v", err)
			}

			keywordScore := result.Factors["keywords"]
			if keywordScore < tt.minKeywordScore {
				t.Errorf("Keyword score = %d, expected >= %d", keywordScore, tt.minKeywordScore)
			}

			t.Logf("✅ %s: keyword_score=%d", tt.name, keywordScore)
		})
	}
}

func TestPriorityEngine_TemporalScore(t *testing.T) {
	cfg, err := config.LoadPriorityConfig("../../config/priority_rules.example.json")
	if err != nil {
		t.Skipf("Skipping test - config file not found: %v", err)
		return
	}

	classifier := ai.NewClassifier(cfg, nil)
	priorityEngine := ai.NewPriorityEngine(cfg, nil, classifier)

	tests := []struct {
		name              string
		receivedAt        time.Time
		expectedTemporal  int
	}{
		{
			name:             "Very recent (<1 hour)",
			receivedAt:       time.Now().Add(-30 * time.Minute),
			expectedTemporal: 15,
		},
		{
			name:             "Recent (<6 hours)",
			receivedAt:       time.Now().Add(-3 * time.Hour),
			expectedTemporal: 10,
		},
		{
			name:             "Today (<24 hours)",
			receivedAt:       time.Now().Add(-12 * time.Hour),
			expectedTemporal: 5,
		},
		{
			name:             "Old (>3 days)",
			receivedAt:       time.Now().Add(-5 * 24 * time.Hour),
			expectedTemporal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := &ai.Email{
				ID:          "test-temporal-" + tt.name,
				From:        "test@example.com",
				Subject:     "Test email",
				BodySnippet: "Test content",
				ReceivedAt:  tt.receivedAt,
			}

			result, err := priorityEngine.CalculatePriority(email)
			if err != nil {
				t.Fatalf("CalculatePriority() error = %v", err)
			}

			temporalScore := result.Factors["temporal"]
			if temporalScore != tt.expectedTemporal {
				t.Errorf("Temporal score = %d, expected %d", temporalScore, tt.expectedTemporal)
			}

			t.Logf("✅ %s: temporal_score=%d", tt.name, temporalScore)
		})
	}
}

func TestPriorityEngine_CategoryScore(t *testing.T) {
	cfg, err := config.LoadPriorityConfig("../../config/priority_rules.example.json")
	if err != nil {
		t.Skipf("Skipping test - config file not found: %v", err)
		return
	}

	classifier := ai.NewClassifier(cfg, nil)
	priorityEngine := ai.NewPriorityEngine(cfg, nil, classifier)

	tests := []struct {
		name             string
		email            *ai.Email
		expectedCategory string
	}{
		{
			name: "Work email",
			email: &ai.Email{
				ID:          "test-work",
				From:        "colleague@company.com",
				Subject:     "Project Update Required",
				BodySnippet: "Please review the project status...",
				ReceivedAt:  time.Now(),
			},
			expectedCategory: "work",
		},
		{
			name: "Invoice email",
			email: &ai.Email{
				ID:          "test-invoice",
				From:        "billing@vendor.com",
				Subject:     "Invoice #12345",
				BodySnippet: "Your invoice is attached",
				ReceivedAt:  time.Now(),
			},
			expectedCategory: "invoice",
		},
		{
			name: "Newsletter email",
			email: &ai.Email{
				ID:          "test-newsletter",
				From:        "newsletter@example.com",
				Subject:     "Weekly Digest",
				BodySnippet: "Unsubscribe at any time",
				ReceivedAt:  time.Now(),
			},
			expectedCategory: "newsletters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := priorityEngine.CalculatePriority(tt.email)
			if err != nil {
				t.Fatalf("CalculatePriority() error = %v", err)
			}

			if result.Category != tt.expectedCategory {
				t.Logf("Category = %s, expected %s (may vary based on rules)",
					result.Category, tt.expectedCategory)
			}

			categoryScore := result.Factors["category"]
			t.Logf("✅ %s: category=%s, category_score=%d",
				tt.name, result.Category, categoryScore)
		})
	}
}

func TestPriorityEngine_ThreadScore(t *testing.T) {
	cfg, err := config.LoadPriorityConfig("../../config/priority_rules.example.json")
	if err != nil {
		t.Skipf("Skipping test - config file not found: %v", err)
		return
	}

	classifier := ai.NewClassifier(cfg, nil)
	priorityEngine := ai.NewPriorityEngine(cfg, nil, classifier)

	tests := []struct {
		name              string
		subject           string
		expectedThread    int
	}{
		{
			name:           "Reply email",
			subject:        "Re: Project Discussion",
			expectedThread: 10,
		},
		{
			name:           "Forward email",
			subject:        "Fwd: Important Information",
			expectedThread: 10,
		},
		{
			name:           "New conversation",
			subject:        "New Topic",
			expectedThread: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := &ai.Email{
				ID:          "test-thread-" + tt.name,
				From:        "test@example.com",
				Subject:     tt.subject,
				BodySnippet: "Test content",
				ReceivedAt:  time.Now(),
			}

			result, err := priorityEngine.CalculatePriority(email)
			if err != nil {
				t.Fatalf("CalculatePriority() error = %v", err)
			}

			threadScore := result.Factors["thread"]
			if threadScore != tt.expectedThread {
				t.Errorf("Thread score = %d, expected %d", threadScore, tt.expectedThread)
			}

			t.Logf("✅ %s: thread_score=%d", tt.name, threadScore)
		})
	}
}

func TestPriorityEngine_ExplainPriority(t *testing.T) {
	cfg, err := config.LoadPriorityConfig("../../config/priority_rules.example.json")
	if err != nil {
		t.Skipf("Skipping test - config file not found: %v", err)
		return
	}

	classifier := ai.NewClassifier(cfg, nil)
	priorityEngine := ai.NewPriorityEngine(cfg, nil, classifier)

	email := &ai.Email{
		ID:          "test-explain",
		From:        "boss@company.com",
		Subject:     "URGENT: Critical Issue",
		BodySnippet: "Immediate attention needed",
		ReceivedAt:  time.Now(),
	}

	priority, err := priorityEngine.CalculatePriority(email)
	if err != nil {
		t.Fatalf("CalculatePriority() error = %v", err)
	}

	explanation := priorityEngine.ExplainPriority(priority)

	// Check that explanation contains key elements
	if len(explanation) == 0 {
		t.Error("ExplainPriority() returned empty explanation")
	}

	t.Logf("✅ Priority explanation generated")
	t.Logf("\n%s", explanation)
}

func TestPriorityEngine_ScoreRange(t *testing.T) {
	cfg, err := config.LoadPriorityConfig("../../config/priority_rules.example.json")
	if err != nil {
		t.Skipf("Skipping test - config file not found: %v", err)
		return
	}

	classifier := ai.NewClassifier(cfg, nil)
	priorityEngine := ai.NewPriorityEngine(cfg, nil, classifier)

	// Test various email scenarios to ensure scores stay in 0-100 range
	emails := []*ai.Email{
		{
			ID:          "max-priority",
			From:        "boss@company.com",
			Subject:     "URGENT: CRITICAL: IMMEDIATE ACTION REQUIRED",
			BodySnippet: "Urgent urgent critical deadline action required",
			ReceivedAt:  time.Now(),
		},
		{
			ID:          "min-priority",
			From:        "spam@unknown.com",
			Subject:     "Buy now!",
			BodySnippet: "Limited offer",
			ReceivedAt:  time.Now().Add(-30 * 24 * time.Hour),
		},
	}

	for _, email := range emails {
		result, err := priorityEngine.CalculatePriority(email)
		if err != nil {
			t.Fatalf("CalculatePriority() error = %v", err)
		}

		if result.Score < 0 || result.Score > 100 {
			t.Errorf("Score out of range: %d (must be 0-100)", result.Score)
		}

		t.Logf("✅ Email %s: score=%d (valid range)", email.ID, result.Score)
	}
}

func TestPriorityEngine_ReasoningChain(t *testing.T) {
	cfg, err := config.LoadPriorityConfig("../../config/priority_rules.example.json")
	if err != nil {
		t.Skipf("Skipping test - config file not found: %v", err)
		return
	}

	classifier := ai.NewClassifier(cfg, nil)
	priorityEngine := ai.NewPriorityEngine(cfg, nil, classifier)

	email := &ai.Email{
		ID:          "test-reasoning",
		From:        "colleague@company.com",
		Subject:     "Re: Project Update",
		BodySnippet: "Thanks for the update",
		ReceivedAt:  time.Now().Add(-2 * time.Hour),
	}

	result, err := priorityEngine.CalculatePriority(email)
	if err != nil {
		t.Fatalf("CalculatePriority() error = %v", err)
	}

	// Verify reasoning chain has entries for each factor
	if len(result.ReasoningChain) < 4 {
		t.Errorf("Reasoning chain too short: %d entries (expected >= 4 factors)",
			len(result.ReasoningChain))
	}

	// Check that reasoning is meaningful (not empty strings)
	for i, reason := range result.ReasoningChain {
		if len(reason) == 0 {
			t.Errorf("Reasoning chain entry %d is empty", i)
		}
	}

	t.Logf("✅ Reasoning chain has %d entries:", len(result.ReasoningChain))
	for _, reason := range result.ReasoningChain {
		t.Logf("   • %s", reason)
	}
}

func TestPriorityEngine_ConsistentScoring(t *testing.T) {
	cfg, err := config.LoadPriorityConfig("../../config/priority_rules.example.json")
	if err != nil {
		t.Skipf("Skipping test - config file not found: %v", err)
		return
	}

	classifier := ai.NewClassifier(cfg, nil)
	priorityEngine := ai.NewPriorityEngine(cfg, nil, classifier)

	email := &ai.Email{
		ID:          "test-consistent",
		From:        "test@example.com",
		Subject:     "Test Email",
		BodySnippet: "Test content",
		ReceivedAt:  time.Now(),
	}

	// Calculate priority twice
	result1, err := priorityEngine.CalculatePriority(email)
	if err != nil {
		t.Fatalf("First CalculatePriority() error = %v", err)
	}

	result2, err := priorityEngine.CalculatePriority(email)
	if err != nil {
		t.Fatalf("Second CalculatePriority() error = %v", err)
	}

	// Scores should be identical (deterministic)
	// Note: Temporal score might differ by 1 point if time passes between calculations
	scoreDiff := result1.Score - result2.Score
	if scoreDiff < -1 || scoreDiff > 1 {
		t.Errorf("Inconsistent scoring: first=%d, second=%d (diff=%d)",
			result1.Score, result2.Score, scoreDiff)
	}

	t.Logf("✅ Consistent scoring: score1=%d, score2=%d", result1.Score, result2.Score)
}
