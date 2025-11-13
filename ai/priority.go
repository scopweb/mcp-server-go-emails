package ai

import (
	"email-mcp-server/config"
	"email-mcp-server/storage"
	"fmt"
	"strings"
	"time"
)

// PriorityEngine calculates email priority scores
type PriorityEngine struct {
	config     *config.PriorityConfig
	db         *storage.Database
	classifier *Classifier
}

// PriorityScore represents the priority score of an email
type PriorityScore struct {
	EmailID        string
	Score          int                // 0-100
	Factors        map[string]int     // Factor name -> contribution
	ReasoningChain []string           // Explanation of scoring
	Category       string
	Timestamp      time.Time
}

// PriorityFactors breaks down the scoring components
type PriorityFactors struct {
	SenderScore     int  // 0-30
	KeywordScore    int  // 0-20
	TemporalScore   int  // 0-15
	CategoryScore   int  // 0-15
	EngagementScore int  // 0-10
	ThreadScore     int  // 0-10
}

// NewPriorityEngine creates a new priority engine
func NewPriorityEngine(cfg *config.PriorityConfig, db *storage.Database, classifier *Classifier) *PriorityEngine {
	return &PriorityEngine{
		config:     cfg,
		db:         db,
		classifier: classifier,
	}
}

// CalculatePriority calculates the priority score for an email
func (pe *PriorityEngine) CalculatePriority(email *Email) (*PriorityScore, error) {
	score := 0
	factors := make(map[string]int)
	reasoning := []string{}

	// Factor 1: Sender Analysis (0-30 points)
	senderScore, senderReasoning := pe.calculateSenderScore(email)
	score += senderScore
	factors["sender"] = senderScore
	reasoning = append(reasoning, senderReasoning...)

	// Factor 2: Urgent Keywords (0-20 points)
	keywordScore, keywordReasoning := pe.calculateKeywordScore(email)
	score += keywordScore
	factors["keywords"] = keywordScore
	reasoning = append(reasoning, keywordReasoning...)

	// Factor 3: Temporal Relevance (0-15 points)
	temporalScore, temporalReasoning := pe.calculateTemporalScore(email)
	score += temporalScore
	factors["temporal"] = temporalScore
	reasoning = append(reasoning, temporalReasoning...)

	// Factor 4: Category Priority (0-15 points) - requires classification first
	categoryScore, categoryReasoning, category := pe.calculateCategoryScore(email)
	score += categoryScore
	factors["category"] = categoryScore
	reasoning = append(reasoning, categoryReasoning...)

	// Factor 5: Engagement History (0-10 points)
	engagementScore, engagementReasoning := pe.calculateEngagementScore(email)
	score += engagementScore
	factors["engagement"] = engagementScore
	reasoning = append(reasoning, engagementReasoning...)

	// Factor 6: Thread Importance (0-10 points)
	threadScore, threadReasoning := pe.calculateThreadScore(email)
	score += threadScore
	factors["thread"] = threadScore
	reasoning = append(reasoning, threadReasoning...)

	// Apply time decay if enabled
	if pe.config.PriorityRules.TimeDecay.Enabled {
		decayedScore, decayReasoning := pe.applyTimeDecay(score, email.ReceivedAt)
		if decayedScore != score {
			reasoning = append(reasoning, decayReasoning)
			score = decayedScore
		}
	}

	// Normalize to 0-100
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return &PriorityScore{
		EmailID:        email.ID,
		Score:          score,
		Factors:        factors,
		ReasoningChain: reasoning,
		Category:       category,
		Timestamp:      time.Now(),
	}, nil
}

// calculateSenderScore evaluates sender importance (0-30 points)
func (pe *PriorityEngine) calculateSenderScore(email *Email) (int, []string) {
	score := 0
	reasoning := []string{}

	// Check VIP senders (30 points)
	if pe.config.IsVIPSender(email.From) {
		score += 30
		reasoning = append(reasoning, "✅ VIP sender (+30)")
		return score, reasoning
	}

	// Check important domains (20 points)
	domain := config.ExtractDomain(email.From)
	if pe.config.IsImportantDomain(domain) {
		score += 20
		reasoning = append(reasoning, fmt.Sprintf("✅ Important domain: %s (+20)", domain))
		return score, reasoning
	}

	// Check sender analytics (0-15 points)
	if analytics, err := pe.db.GetSenderAnalytics(email.From); err == nil {
		if analytics.IsVIP {
			score += 25
			reasoning = append(reasoning, "✅ Learned VIP sender (+25)")
		} else {
			// Base score on engagement
			engagementBonus := (analytics.EngagementScore * 15) / 100
			if engagementBonus > 0 {
				score += engagementBonus
				reasoning = append(reasoning, fmt.Sprintf("📊 Engagement score: %d (+%d)", analytics.EngagementScore, engagementBonus))
			}
		}
	}

	if score == 0 {
		reasoning = append(reasoning, "👤 Unknown sender (+0)")
	}

	return score, reasoning
}

// calculateKeywordScore analyzes urgent keywords (0-20 points)
func (pe *PriorityEngine) calculateKeywordScore(email *Email) (int, []string) {
	score := 0
	reasoning := []string{}

	// Check subject for urgent keywords
	hasUrgent, keyword := pe.config.HasUrgentKeyword(email.Subject)
	if hasUrgent {
		score += 20
		reasoning = append(reasoning, fmt.Sprintf("🚨 Urgent keyword in subject: '%s' (+20)", keyword))
		return score, reasoning
	}

	// Check body snippet for urgent keywords
	hasUrgent, keyword = pe.config.HasUrgentKeyword(email.BodySnippet)
	if hasUrgent {
		score += 15
		reasoning = append(reasoning, fmt.Sprintf("⚠️  Urgent keyword in body: '%s' (+15)", keyword))
		return score, reasoning
	}

	// Check for action-required keywords
	actionKeywords := []string{"action required", "please respond", "response needed", "deadline", "due date"}
	for _, kw := range actionKeywords {
		if containsIgnoreCase(email.Subject, kw) || containsIgnoreCase(email.BodySnippet, kw) {
			score += 10
			reasoning = append(reasoning, fmt.Sprintf("📋 Action keyword: '%s' (+10)", kw))
			return score, reasoning
		}
	}

	reasoning = append(reasoning, "📝 No urgent keywords (+0)")
	return score, reasoning
}

// calculateTemporalScore evaluates time relevance (0-15 points)
func (pe *PriorityEngine) calculateTemporalScore(email *Email) (int, []string) {
	score := 0
	reasoning := []string{}

	age := time.Since(email.ReceivedAt)

	if age < 1*time.Hour {
		score = 15
		reasoning = append(reasoning, "⏰ Very recent: <1 hour (+15)")
	} else if age < 6*time.Hour {
		score = 10
		reasoning = append(reasoning, "⏱️  Recent: <6 hours (+10)")
	} else if age < 24*time.Hour {
		score = 5
		reasoning = append(reasoning, "📅 Today (+5)")
	} else if age < 3*24*time.Hour {
		score = 2
		reasoning = append(reasoning, "📆 Last 3 days (+2)")
	} else {
		reasoning = append(reasoning, fmt.Sprintf("🕰️  Old: %d days (+0)", int(age.Hours()/24)))
	}

	return score, reasoning
}

// calculateCategoryScore evaluates category importance (0-15 points)
func (pe *PriorityEngine) calculateCategoryScore(email *Email) (int, []string, string) {
	score := 0
	reasoning := []string{}
	category := "unknown"

	// Get or calculate classification
	var classification *ClassificationResult
	var err error

	// Try to get from database first
	if pe.classifier != nil {
		classification, err = pe.classifier.GetClassification(email.ID)
		if err != nil {
			// Classify now
			classification, _ = pe.classifier.Classify(email)
		}
	}

	if classification != nil {
		category = classification.Category
		categoryBoost := pe.config.GetCategoryPriority(category)

		// Normalize category boost to 0-15 range
		normalizedScore := categoryBoost
		if normalizedScore > 15 {
			normalizedScore = 15
		}
		if normalizedScore < 0 {
			normalizedScore = 0
		}

		score = normalizedScore
		if categoryBoost > 0 {
			reasoning = append(reasoning, fmt.Sprintf("📁 Category '%s' (+%d)", category, score))
		} else if categoryBoost < 0 {
			reasoning = append(reasoning, fmt.Sprintf("📁 Category '%s' (%d)", category, categoryBoost))
		} else {
			reasoning = append(reasoning, fmt.Sprintf("📁 Category '%s' (+0)", category))
		}
	} else {
		reasoning = append(reasoning, "📁 Unknown category (+0)")
	}

	return score, reasoning, category
}

// calculateEngagementScore evaluates historical engagement (0-10 points)
func (pe *PriorityEngine) calculateEngagementScore(email *Email) (int, []string) {
	score := 0
	reasoning := []string{}

	analytics, err := pe.db.GetSenderAnalytics(email.From)
	if err != nil {
		reasoning = append(reasoning, "📊 No engagement history (+0)")
		return score, reasoning
	}

	// Calculate engagement based on read/reply rates
	if analytics.TotalEmails > 0 {
		readRate := float64(analytics.ReadCount) / float64(analytics.TotalEmails)
		replyRate := float64(analytics.ReplyCount) / float64(analytics.TotalEmails)

		engagementScore := int((readRate*5 + replyRate*5))
		if engagementScore > 10 {
			engagementScore = 10
		}

		score = engagementScore

		if score > 5 {
			reasoning = append(reasoning, fmt.Sprintf("💬 High engagement: %.0f%% read, %.0f%% reply (+%d)",
				readRate*100, replyRate*100, score))
		} else if score > 0 {
			reasoning = append(reasoning, fmt.Sprintf("📬 Some engagement (+%d)", score))
		}
	}

	if score == 0 {
		reasoning = append(reasoning, "📊 No engagement history (+0)")
	}

	return score, reasoning
}

// calculateThreadScore evaluates thread importance (0-10 points)
func (pe *PriorityEngine) calculateThreadScore(email *Email) (int, []string) {
	score := 0
	reasoning := []string{}

	// Check if email is part of a thread (reply/forward)
	isReply := strings.Contains(strings.ToLower(email.Subject), "re:") ||
		strings.Contains(strings.ToLower(email.Subject), "fwd:")

	if isReply {
		score = 10
		reasoning = append(reasoning, "🔗 Part of active thread (+10)")
	} else {
		reasoning = append(reasoning, "✉️  New conversation (+0)")
	}

	return score, reasoning
}

// applyTimeDecay reduces priority for old emails
func (pe *PriorityEngine) applyTimeDecay(score int, receivedAt time.Time) (int, string) {
	if !pe.config.PriorityRules.TimeDecay.Enabled {
		return score, ""
	}

	age := time.Since(receivedAt)
	maxAge := time.Duration(pe.config.PriorityRules.TimeDecay.MaxAgeHours) * time.Hour

	if age > maxAge {
		decayRate := pe.config.PriorityRules.TimeDecay.DecayRate
		hoursOld := age.Hours()
		decayFactor := 1.0 - (decayRate * (hoursOld / 24.0))

		if decayFactor < 0 {
			decayFactor = 0
		}

		newScore := int(float64(score) * decayFactor)
		reduction := score - newScore

		return newScore, fmt.Sprintf("⏳ Time decay: -%d (age: %.1f days)", reduction, hoursOld/24)
	}

	return score, ""
}

// SavePriority saves a priority score to the database
func (pe *PriorityEngine) SavePriority(priorityScore *PriorityScore) error {
	priority := &storage.Priority{
		EmailID:   priorityScore.EmailID,
		Score:     priorityScore.Score,
		Factors:   priorityScore.Factors,
		Reasoning: strings.Join(priorityScore.ReasoningChain, "; "),
	}

	return pe.db.SavePriority(priority)
}

// GetPriorityEmails retrieves top priority emails
func (pe *PriorityEngine) GetPriorityEmails(accountID string, minScore, limit int) ([]*storage.Email, error) {
	return pe.db.GetPriorityEmails(accountID, minScore, limit)
}

// RecalculatePriorities recalculates priorities for all emails in an account
func (pe *PriorityEngine) RecalculatePriorities(accountID string) error {
	// Get all emails for the account
	filter := storage.EmailFilter{
		AccountID: accountID,
		Limit:     1000, // Process in batches
	}

	emails, err := pe.db.ListEmails(filter)
	if err != nil {
		return fmt.Errorf("failed to list emails: %w", err)
	}

	// Process each email
	for _, dbEmail := range emails {
		email := &Email{
			ID:          dbEmail.ID,
			From:        dbEmail.From,
			To:          dbEmail.To,
			Subject:     dbEmail.Subject,
			BodySnippet: dbEmail.BodySnippet,
			ReceivedAt:  dbEmail.ReceivedAt,
		}

		priority, err := pe.CalculatePriority(email)
		if err != nil {
			return fmt.Errorf("failed to calculate priority for email %s: %w", email.ID, err)
		}

		if err := pe.SavePriority(priority); err != nil {
			return fmt.Errorf("failed to save priority for email %s: %w", email.ID, err)
		}
	}

	return nil
}

// AnalyzePriorityDistribution returns statistics about priority distribution
func (pe *PriorityEngine) AnalyzePriorityDistribution(accountID string) (map[string]int, error) {
	distribution := map[string]int{
		"critical": 0,  // 90-100
		"high":     0,  // 70-89
		"medium":   0,  // 40-69
		"low":      0,  // 20-39
		"minimal":  0,  // 0-19
	}

	// This would query the database for priority distribution
	// For now, return empty distribution
	return distribution, nil
}

// GetStats returns priority engine statistics
func (pe *PriorityEngine) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"vip_senders_count":       len(pe.config.PriorityRules.VIPSenders),
		"important_domains_count": len(pe.config.PriorityRules.ImportantDomains),
		"urgent_keywords_count":   len(pe.config.PriorityRules.UrgentKeywords),
		"time_decay_enabled":      pe.config.PriorityRules.TimeDecay.Enabled,
	}
}

// UpdateVIPStatus marks a sender as VIP
func (pe *PriorityEngine) UpdateVIPStatus(senderEmail string, isVIP bool) error {
	analytics, err := pe.db.GetSenderAnalytics(senderEmail)
	if err != nil {
		// Create new analytics entry
		analytics = &storage.SenderAnalytics{
			EmailAddress: senderEmail,
			IsVIP:        isVIP,
		}
	} else {
		analytics.IsVIP = isVIP
	}

	return pe.db.UpdateSenderAnalytics(analytics)
}

// GetPriorityBreakdown returns a detailed breakdown of a priority score
func (pe *PriorityEngine) GetPriorityBreakdown(emailID string) (*PriorityScore, error) {
	// Get priority from database
	priority, err := pe.db.GetPriority(emailID)
	if err != nil {
		return nil, err
	}

	reasoningParts := strings.Split(priority.Reasoning, "; ")

	return &PriorityScore{
		EmailID:        priority.EmailID,
		Score:          priority.Score,
		Factors:        priority.Factors,
		ReasoningChain: reasoningParts,
		Timestamp:      priority.CalculatedAt,
	}, nil
}

// ExplainPriority provides a human-readable explanation of an email's priority
func (pe *PriorityEngine) ExplainPriority(priority *PriorityScore) string {
	var explanation strings.Builder

	explanation.WriteString(fmt.Sprintf("Priority Score: %d/100\n\n", priority.Score))

	if priority.Score >= 90 {
		explanation.WriteString("🔴 CRITICAL PRIORITY\n")
	} else if priority.Score >= 70 {
		explanation.WriteString("🟠 HIGH PRIORITY\n")
	} else if priority.Score >= 40 {
		explanation.WriteString("🟡 MEDIUM PRIORITY\n")
	} else if priority.Score >= 20 {
		explanation.WriteString("🟢 LOW PRIORITY\n")
	} else {
		explanation.WriteString("⚪ MINIMAL PRIORITY\n")
	}

	explanation.WriteString("\nScore Breakdown:\n")
	for _, reason := range priority.ReasoningChain {
		explanation.WriteString(fmt.Sprintf("  • %s\n", reason))
	}

	return explanation.String()
}
