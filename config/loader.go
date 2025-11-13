package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sync"
)

// PriorityConfig holds all priority and classification rules
type PriorityConfig struct {
	Version        string                      `json:"version"`
	PriorityRules  PriorityRules               `json:"priority_rules"`
	ClassificationRules map[string]ClassificationRule `json:"classification_rules"`
	Learning       LearningConfig              `json:"learning"`
	Notifications  NotificationConfig          `json:"notifications"`
	mu             sync.RWMutex
}

// PriorityRules defines sender and keyword-based priority rules
type PriorityRules struct {
	VIPSenders        []string          `json:"vip_senders"`
	ImportantDomains  []string          `json:"important_domains"`
	UrgentKeywords    []string          `json:"urgent_keywords"`
	IgnoreSenders     []string          `json:"ignore_senders"`
	IgnoreSubjects    []string          `json:"ignore_subjects"`
	CategoryPriority  map[string]int    `json:"category_priority"`
	TimeDecay         TimeDecayConfig   `json:"time_decay"`
}

// TimeDecayConfig configures how priority decreases over time
type TimeDecayConfig struct {
	Enabled      bool    `json:"enabled"`
	MaxAgeHours  int     `json:"max_age_hours"`
	DecayRate    float64 `json:"decay_rate"`
}

// ClassificationRule defines rules for email classification
type ClassificationRule struct {
	Description   string      `json:"description"`
	Conditions    []Condition `json:"conditions"`
	PriorityBoost int         `json:"priority_boost"`
	Confidence    float64     `json:"confidence"`
	Tags          []string    `json:"tags,omitempty"`
}

// Condition represents a matching condition
type Condition struct {
	Field    string   `json:"field"`    // "from", "subject", "body", "headers"
	Operator string   `json:"operator"` // "contains", "contains_any", "regex", "domain_in", "domain_not_in"
	Value    string   `json:"value,omitempty"`
	Values   []string `json:"values,omitempty"`
}

// LearningConfig configures the learning system
type LearningConfig struct {
	Enabled        bool     `json:"enabled"`
	MinSamples     int      `json:"min_samples"`
	AdjustmentRate float64  `json:"adjustment_rate"`
	Features       []string `json:"features"`
}

// NotificationConfig configures notifications
type NotificationConfig struct {
	HighPriorityThreshold int                `json:"high_priority_threshold"`
	CriticalThreshold     int                `json:"critical_threshold"`
	Channels              map[string]Channel `json:"channels"`
}

// Channel represents a notification channel
type Channel struct {
	Enabled    bool     `json:"enabled"`
	Recipients []string `json:"recipients,omitempty"`
	URLs       []string `json:"urls,omitempty"`
}

// AIConfig holds AI-related configuration
type AIConfig struct {
	Provider       string                 `json:"provider"`
	Model          string                 `json:"model"`
	APIKeyEnv      string                 `json:"api_key_env"`
	MaxTokens      int                    `json:"max_tokens"`
	Temperature    float64                `json:"temperature"`
	TimeoutSeconds int                    `json:"timeout_seconds"`
	Classification ClassificationConfig   `json:"classification"`
	Summarization  SummarizationConfig    `json:"summarization"`
}

// ClassificationConfig configures AI classification
type ClassificationConfig struct {
	Enabled             bool              `json:"enabled"`
	FallbackToRules     bool              `json:"fallback_to_rules"`
	ConfidenceThreshold float64           `json:"confidence_threshold"`
	Cache               CacheConfig       `json:"cache"`
	RateLimiting        RateLimitConfig   `json:"rate_limiting"`
	PromptTemplate      string            `json:"prompt_template"`
}

// SummarizationConfig configures AI summarization
type SummarizationConfig struct {
	Enabled      bool                   `json:"enabled"`
	Styles       map[string]StyleConfig `json:"styles"`
	DefaultStyle string                 `json:"default_style"`
	Cache        CacheConfig            `json:"cache"`
}

// StyleConfig defines a summary style
type StyleConfig struct {
	MaxTokens    int     `json:"max_tokens"`
	Temperature  float64 `json:"temperature"`
	Instructions string  `json:"instructions"`
}

// CacheConfig configures caching
type CacheConfig struct {
	Enabled  bool `json:"enabled"`
	TTLHours int  `json:"ttl_hours"`
}

// RateLimitConfig configures rate limiting
type RateLimitConfig struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	Burst             int `json:"burst"`
}

// Global config instances
var (
	priorityConfig *PriorityConfig
	aiConfig       *AIConfig
	configMutex    sync.RWMutex
)

// LoadPriorityConfig loads priority rules from file
func LoadPriorityConfig(path string) (*PriorityConfig, error) {
	configMutex.Lock()
	defer configMutex.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &PriorityConfig{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	priorityConfig = config
	return config, nil
}

// LoadAIConfig loads AI configuration from file
func LoadAIConfig(path string) (*AIConfig, error) {
	configMutex.Lock()
	defer configMutex.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read AI config file: %w", err)
	}

	config := &AIConfig{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse AI config: %w", err)
	}

	aiConfig = config
	return config, nil
}

// GetPriorityConfig returns the current priority configuration
func GetPriorityConfig() *PriorityConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return priorityConfig
}

// GetAIConfig returns the current AI configuration
func GetAIConfig() *AIConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return aiConfig
}

// Validate validates the priority configuration
func (pc *PriorityConfig) Validate() error {
	// Validate category priorities are in valid range
	for category, priority := range pc.PriorityRules.CategoryPriority {
		if priority < -100 || priority > 100 {
			return fmt.Errorf("invalid priority for category %s: %d (must be between -100 and 100)", category, priority)
		}
	}

	// Validate classification rules
	for name, rule := range pc.ClassificationRules {
		if rule.Confidence < 0 || rule.Confidence > 1 {
			return fmt.Errorf("invalid confidence for rule %s: %f (must be between 0 and 1)", name, rule.Confidence)
		}

		// Validate conditions
		for i, cond := range rule.Conditions {
			if err := cond.Validate(); err != nil {
				return fmt.Errorf("invalid condition %d in rule %s: %w", i, name, err)
			}
		}
	}

	return nil
}

// Validate validates a condition
func (c *Condition) Validate() error {
	validFields := map[string]bool{
		"from": true, "subject": true, "body": true, "headers": true, "to": true,
	}
	if !validFields[c.Field] {
		return fmt.Errorf("invalid field: %s", c.Field)
	}

	validOperators := map[string]bool{
		"contains": true, "contains_any": true, "regex": true,
		"domain_in": true, "domain_not_in": true,
	}
	if !validOperators[c.Operator] {
		return fmt.Errorf("invalid operator: %s", c.Operator)
	}

	// Validate regex if operator is regex
	if c.Operator == "regex" && c.Value != "" {
		if _, err := regexp.Compile(c.Value); err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	return nil
}

// IsVIPSender checks if a sender is in the VIP list
func (pc *PriorityConfig) IsVIPSender(email string) bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	for _, vip := range pc.PriorityRules.VIPSenders {
		if vip == email {
			return true
		}
	}
	return false
}

// IsImportantDomain checks if a domain is in the important domains list
func (pc *PriorityConfig) IsImportantDomain(domain string) bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	for _, d := range pc.PriorityRules.ImportantDomains {
		if d == domain {
			return true
		}
	}
	return false
}

// HasUrgentKeyword checks if text contains any urgent keyword
func (pc *PriorityConfig) HasUrgentKeyword(text string) (bool, string) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	textLower := toLower(text)
	for _, keyword := range pc.PriorityRules.UrgentKeywords {
		if contains(textLower, toLower(keyword)) {
			return true, keyword
		}
	}
	return false, ""
}

// ShouldIgnoreSender checks if a sender should be ignored
func (pc *PriorityConfig) ShouldIgnoreSender(email string) bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	emailLower := toLower(email)
	for _, pattern := range pc.PriorityRules.IgnoreSenders {
		if matchPattern(emailLower, toLower(pattern)) {
			return true
		}
	}
	return false
}

// ShouldIgnoreSubject checks if a subject should be ignored
func (pc *PriorityConfig) ShouldIgnoreSubject(subject string) bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	subjectLower := toLower(subject)
	for _, keyword := range pc.PriorityRules.IgnoreSubjects {
		if contains(subjectLower, toLower(keyword)) {
			return true
		}
	}
	return false
}

// GetCategoryPriority returns the priority boost for a category
func (pc *PriorityConfig) GetCategoryPriority(category string) int {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if priority, ok := pc.PriorityRules.CategoryPriority[category]; ok {
		return priority
	}
	return 0
}

// ==================================
// Helper Functions
// ==================================

func toLower(s string) string {
	// Simple ASCII lowercase for now
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func matchPattern(s, pattern string) bool {
	// Simple pattern matching with * wildcard
	if pattern == "*" {
		return true
	}

	// No wildcards - exact match
	if indexOf(pattern, "*") == -1 {
		return s == pattern
	}

	// Pattern starts with *
	if len(pattern) > 0 && pattern[0] == '*' {
		suffix := pattern[1:]
		return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
	}

	// Pattern ends with *
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(s) >= len(prefix) && s[:len(prefix)] == prefix
	}

	// Pattern has * in the middle
	parts := []string{}
	current := ""
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '*' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(pattern[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	// Check if all parts are in order
	pos := 0
	for _, part := range parts {
		idx := indexOf(s[pos:], part)
		if idx == -1 {
			return false
		}
		pos += idx + len(part)
	}

	return true
}

// ExtractDomain extracts domain from email address
func ExtractDomain(email string) string {
	atIdx := indexOf(email, "@")
	if atIdx == -1 {
		return ""
	}
	return email[atIdx+1:]
}
