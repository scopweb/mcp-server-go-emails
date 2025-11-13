package ai

import (
	"crypto/md5"
	"email-mcp-server/config"
	"email-mcp-server/storage"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Classifier handles email classification
type Classifier struct {
	config *config.PriorityConfig
	db     *storage.Database
	cache  map[string]*ClassificationResult
}

// ClassificationResult represents the result of email classification
type ClassificationResult struct {
	EmailID    string
	Category   string
	Confidence float64
	Method     string // "rules" | "ai" | "hybrid"
	Tags       []string
	Reasoning  string
	Timestamp  time.Time
}

// Email represents an email for classification
type Email struct {
	ID           string
	From         string
	To           string
	Subject      string
	Body         string
	BodySnippet  string
	Headers      map[string]string
	ReceivedAt   time.Time
}

// NewClassifier creates a new email classifier
func NewClassifier(cfg *config.PriorityConfig, db *storage.Database) *Classifier {
	return &Classifier{
		config: cfg,
		db:     db,
		cache:  make(map[string]*ClassificationResult),
	}
}

// Classify classifies an email using rules
func (c *Classifier) Classify(email *Email) (*ClassificationResult, error) {
	// Check cache first
	cacheKey := c.getCacheKey(email)
	if cached, ok := c.cache[cacheKey]; ok {
		if time.Since(cached.Timestamp) < 24*time.Hour {
			return cached, nil
		}
	}

	// Check if email should be ignored
	if c.config.ShouldIgnoreSender(email.From) {
		return &ClassificationResult{
			EmailID:    email.ID,
			Category:   "spam",
			Confidence: 0.99,
			Method:     "rules",
			Tags:       []string{"ignored", "auto"},
			Reasoning:  "Sender is in ignore list",
			Timestamp:  time.Now(),
		}, nil
	}

	if c.config.ShouldIgnoreSubject(email.Subject) {
		return &ClassificationResult{
			EmailID:    email.ID,
			Category:   "newsletters",
			Confidence: 0.95,
			Method:     "rules",
			Tags:       []string{"ignored", "auto"},
			Reasoning:  "Subject matches ignore pattern",
			Timestamp:  time.Now(),
		}, nil
	}

	// Try classification rules
	result := c.classifyByRules(email)

	// Cache result
	c.cache[cacheKey] = result

	return result, nil
}

// classifyByRules applies classification rules to an email
func (c *Classifier) classifyByRules(email *Email) *ClassificationResult {
	var bestMatch *ClassificationResult
	highestConfidence := 0.0

	// Try each classification rule
	for category, rule := range c.config.ClassificationRules {
		if c.matchesRule(email, rule) {
			if rule.Confidence > highestConfidence {
				highestConfidence = rule.Confidence
				bestMatch = &ClassificationResult{
					EmailID:    email.ID,
					Category:   category,
					Confidence: rule.Confidence,
					Method:     "rules",
					Tags:       rule.Tags,
					Reasoning:  fmt.Sprintf("Matched rule: %s", rule.Description),
					Timestamp:  time.Now(),
				}
			}
		}
	}

	// If no rule matched, use default classification
	if bestMatch == nil {
		bestMatch = c.defaultClassification(email)
	}

	return bestMatch
}

// matchesRule checks if an email matches a classification rule
func (c *Classifier) matchesRule(email *Email, rule config.ClassificationRule) bool {
	// All conditions must match for the rule to apply
	for _, condition := range rule.Conditions {
		if !c.matchesCondition(email, condition) {
			return false
		}
	}
	return true
}

// matchesCondition checks if an email matches a single condition
func (c *Classifier) matchesCondition(email *Email, cond config.Condition) bool {
	// Get the field value
	var fieldValue string
	switch cond.Field {
	case "from":
		fieldValue = email.From
	case "to":
		fieldValue = email.To
	case "subject":
		fieldValue = email.Subject
	case "body":
		fieldValue = email.BodySnippet
	case "headers":
		fieldValue = c.getAllHeaders(email)
	default:
		return false
	}

	// Apply the operator
	switch cond.Operator {
	case "contains":
		return containsIgnoreCase(fieldValue, cond.Value)

	case "contains_any":
		for _, val := range cond.Values {
			if containsIgnoreCase(fieldValue, val) {
				return true
			}
		}
		return false

	case "regex":
		re, err := regexp.Compile(cond.Value)
		if err != nil {
			return false
		}
		return re.MatchString(fieldValue)

	case "domain_in":
		domain := config.ExtractDomain(fieldValue)
		for _, allowedDomain := range cond.Values {
			if domain == allowedDomain {
				return true
			}
		}
		return false

	case "domain_not_in":
		domain := config.ExtractDomain(fieldValue)
		for _, excludedDomain := range cond.Values {
			if domain == excludedDomain {
				return false
			}
		}
		return true

	default:
		return false
	}
}

// defaultClassification provides a default classification when no rules match
func (c *Classifier) defaultClassification(email *Email) *ClassificationResult {
	category := "personal"
	reasoning := "Default classification - no specific rules matched"
	confidence := 0.5

	// Basic heuristics
	subjectLower := strings.ToLower(email.Subject)
	fromLower := strings.ToLower(email.From)

	// Check for common patterns
	if containsAny(fromLower, []string{"noreply@", "no-reply@", "notifications@"}) {
		category = "newsletters"
		reasoning = "Automated sender detected"
		confidence = 0.7
	} else if containsAny(subjectLower, []string{"newsletter", "subscription", "digest"}) {
		category = "newsletters"
		reasoning = "Newsletter keywords in subject"
		confidence = 0.75
	} else if containsAny(subjectLower, []string{"sale", "offer", "discount", "% off", "deal"}) {
		category = "promotions"
		reasoning = "Promotional keywords detected"
		confidence = 0.80
	} else if containsAny(subjectLower, []string{"invoice", "payment", "receipt", "bill"}) {
		category = "invoice"
		reasoning = "Financial keywords detected"
		confidence = 0.85
	} else if containsAny(subjectLower, []string{"urgent", "asap", "important", "critical"}) {
		category = "urgent"
		reasoning = "Urgent keywords detected"
		confidence = 0.90
	}

	return &ClassificationResult{
		EmailID:    email.ID,
		Category:   category,
		Confidence: confidence,
		Method:     "rules",
		Tags:       []string{"default", "heuristic"},
		Reasoning:  reasoning,
		Timestamp:  time.Now(),
	}
}

// ClassifyBatch classifies multiple emails
func (c *Classifier) ClassifyBatch(emails []*Email) ([]*ClassificationResult, error) {
	results := make([]*ClassificationResult, len(emails))

	for i, email := range emails {
		result, err := c.Classify(email)
		if err != nil {
			return nil, fmt.Errorf("failed to classify email %s: %w", email.ID, err)
		}
		results[i] = result
	}

	return results, nil
}

// SaveClassification saves a classification to the database
func (c *Classifier) SaveClassification(result *ClassificationResult) error {
	classification := &storage.Classification{
		EmailID:    result.EmailID,
		Category:   result.Category,
		Confidence: result.Confidence,
		Method:     result.Method,
		Tags:       result.Tags,
		Reasoning:  result.Reasoning,
	}

	return c.db.SaveClassification(classification)
}

// GetClassification retrieves a classification from the database
func (c *Classifier) GetClassification(emailID string) (*ClassificationResult, error) {
	classification, err := c.db.GetClassification(emailID)
	if err != nil {
		return nil, err
	}

	return &ClassificationResult{
		EmailID:    classification.EmailID,
		Category:   classification.Category,
		Confidence: classification.Confidence,
		Method:     classification.Method,
		Tags:       classification.Tags,
		Reasoning:  classification.Reasoning,
		Timestamp:  classification.ClassifiedAt,
	}, nil
}

// LearnFromFeedback updates classification based on user feedback
func (c *Classifier) LearnFromFeedback(emailID, correctCategory string) error {
	// TODO: Implement learning logic
	// For now, just update the classification
	result := &ClassificationResult{
		EmailID:    emailID,
		Category:   correctCategory,
		Confidence: 1.0,
		Method:     "user",
		Tags:       []string{"user_corrected"},
		Reasoning:  "User feedback",
		Timestamp:  time.Now(),
	}

	return c.SaveClassification(result)
}

// GetStats returns classification statistics
func (c *Classifier) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"cache_size":    len(c.cache),
		"rules_count":   len(c.config.ClassificationRules),
		"categories":    c.getCategories(),
	}

	return stats
}

// ==================================
// Helper Functions
// ==================================

// getCacheKey generates a cache key for an email
func (c *Classifier) getCacheKey(email *Email) string {
	data := fmt.Sprintf("%s:%s:%s", email.From, email.Subject, email.ReceivedAt.Format(time.RFC3339))
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// getAllHeaders concatenates all header values
func (c *Classifier) getAllHeaders(email *Email) string {
	var headers []string
	for key, value := range email.Headers {
		headers = append(headers, fmt.Sprintf("%s: %s", key, value))
	}
	return strings.Join(headers, "\n")
}

// getCategories returns list of all known categories
func (c *Classifier) getCategories() []string {
	categories := make([]string, 0, len(c.config.ClassificationRules))
	for category := range c.config.ClassificationRules {
		categories = append(categories, category)
	}
	// Add default categories
	defaultCategories := []string{"personal", "spam", "newsletters", "promotions", "invoice", "urgent"}
	for _, cat := range defaultCategories {
		found := false
		for _, existing := range categories {
			if existing == cat {
				found = true
				break
			}
		}
		if !found {
			categories = append(categories, cat)
		}
	}
	return categories
}

// containsIgnoreCase checks if s contains substr (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// containsAny checks if s contains any of the substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// Categories returns all supported categories
func (c *Classifier) Categories() []string {
	return c.getCategories()
}

// ClearCache clears the classification cache
func (c *Classifier) ClearCache() {
	c.cache = make(map[string]*ClassificationResult)
}
