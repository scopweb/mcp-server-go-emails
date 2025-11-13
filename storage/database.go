package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	// SQLite driver - Install with: go get modernc.org/sqlite@v1.28.0
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

// Database manages the SQLite database connection
type Database struct {
	db     *sql.DB
	dbPath string
	mu     sync.RWMutex
}

// Config holds database configuration
type Config struct {
	Path            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DefaultConfig returns default database configuration
func DefaultConfig() *Config {
	return &Config{
		Path:            "./data/emails.db",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	}
}

// New creates a new Database instance
func New(config *Config) (*Database, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Ensure directory exists
	dir := filepath.Dir(config.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	// NOTE: Uncomment when modernc.org/sqlite is available
	/*
	db, err := sql.Open("sqlite", config.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	*/

	database := &Database{
		// db:     db,
		dbPath: config.Path,
	}

	// Initialize schema
	// if err := database.initSchema(); err != nil {
	// 	return nil, fmt.Errorf("failed to initialize schema: %w", err)
	// }

	return database, nil
}

// initSchema initializes the database schema
func (d *Database) initSchema() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Read schema from embedded file
	schemaSQL, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Execute schema
	if _, err := d.db.Exec(string(schemaSQL)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// Begin starts a new transaction
func (d *Database) Begin() (*sql.Tx, error) {
	return d.db.Begin()
}

// ==================================
// Email Operations
// ==================================

// Email represents an email message
type Email struct {
	ID           string
	AccountID    string
	MessageID    string
	ThreadID     string
	From         string
	To           string
	Subject      string
	BodySnippet  string
	ReceivedAt   time.Time
	Read         bool
	Starred      bool
	Deleted      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Related data (loaded separately)
	Classification *Classification
	Priority       *Priority
}

// CreateEmail inserts a new email into the database
func (d *Database) CreateEmail(email *Email) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `
		INSERT INTO emails (
			id, account_id, message_id, thread_id, from_addr, to_addr,
			subject, body_snippet, received_at, read, starred, deleted
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query,
		email.ID, email.AccountID, email.MessageID, email.ThreadID,
		email.From, email.To, email.Subject, email.BodySnippet,
		email.ReceivedAt, email.Read, email.Starred, email.Deleted,
	)

	return err
}

// GetEmail retrieves an email by ID
func (d *Database) GetEmail(id string) (*Email, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, account_id, message_id, thread_id, from_addr, to_addr,
			subject, body_snippet, received_at, read, starred, deleted,
			created_at, updated_at
		FROM emails
		WHERE id = ? AND deleted = 0
	`

	email := &Email{}
	err := d.db.QueryRow(query, id).Scan(
		&email.ID, &email.AccountID, &email.MessageID, &email.ThreadID,
		&email.From, &email.To, &email.Subject, &email.BodySnippet,
		&email.ReceivedAt, &email.Read, &email.Starred, &email.Deleted,
		&email.CreatedAt, &email.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("email not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	return email, nil
}

// EmailFilter defines filters for listing emails
type EmailFilter struct {
	AccountID   string
	Category    string
	MinPriority int
	Read        *bool
	Starred     *bool
	DateFrom    time.Time
	DateTo      time.Time
	Limit       int
	Offset      int
}

// ListEmails retrieves emails based on filters
func (d *Database) ListEmails(filter EmailFilter) ([]*Email, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT e.id, e.account_id, e.message_id, e.thread_id, e.from_addr, e.to_addr,
			e.subject, e.body_snippet, e.received_at, e.read, e.starred, e.deleted,
			e.created_at, e.updated_at
		FROM emails e
		WHERE e.deleted = 0
	`

	args := []interface{}{}

	if filter.AccountID != "" {
		query += " AND e.account_id = ?"
		args = append(args, filter.AccountID)
	}

	if filter.Category != "" {
		query += ` AND e.id IN (
			SELECT email_id FROM classifications WHERE category = ?
		)`
		args = append(args, filter.Category)
	}

	if filter.MinPriority > 0 {
		query += ` AND e.id IN (
			SELECT email_id FROM priorities WHERE score >= ?
		)`
		args = append(args, filter.MinPriority)
	}

	if filter.Read != nil {
		query += " AND e.read = ?"
		args = append(args, *filter.Read)
	}

	if filter.Starred != nil {
		query += " AND e.starred = ?"
		args = append(args, *filter.Starred)
	}

	if !filter.DateFrom.IsZero() {
		query += " AND e.received_at >= ?"
		args = append(args, filter.DateFrom)
	}

	if !filter.DateTo.IsZero() {
		query += " AND e.received_at <= ?"
		args = append(args, filter.DateTo)
	}

	query += " ORDER BY e.received_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emails := []*Email{}
	for rows.Next() {
		email := &Email{}
		err := rows.Scan(
			&email.ID, &email.AccountID, &email.MessageID, &email.ThreadID,
			&email.From, &email.To, &email.Subject, &email.BodySnippet,
			&email.ReceivedAt, &email.Read, &email.Starred, &email.Deleted,
			&email.CreatedAt, &email.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}

	return emails, rows.Err()
}

// UpdateEmail updates an existing email
func (d *Database) UpdateEmail(email *Email) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `
		UPDATE emails
		SET read = ?, starred = ?, deleted = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := d.db.Exec(query, email.Read, email.Starred, email.Deleted, email.ID)
	return err
}

// ==================================
// Classification Operations
// ==================================

// Classification represents email classification
type Classification struct {
	EmailID      string
	Category     string
	Confidence   float64
	Method       string
	Tags         []string
	Reasoning    string
	ClassifiedAt time.Time
}

// SaveClassification saves email classification
func (d *Database) SaveClassification(c *Classification) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Convert tags to JSON
	tagsJSON := "[]"
	if len(c.Tags) > 0 {
		tagsJSON = fmt.Sprintf(`["%s"]`, c.Tags[0]) // Simplified for now
	}

	query := `
		INSERT OR REPLACE INTO classifications (
			email_id, category, confidence, method, tags, reasoning
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query,
		c.EmailID, c.Category, c.Confidence, c.Method, tagsJSON, c.Reasoning,
	)

	return err
}

// GetClassification retrieves classification for an email
func (d *Database) GetClassification(emailID string) (*Classification, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT email_id, category, confidence, method, tags, reasoning, classified_at
		FROM classifications
		WHERE email_id = ?
	`

	c := &Classification{}
	var tagsJSON string

	err := d.db.QueryRow(query, emailID).Scan(
		&c.EmailID, &c.Category, &c.Confidence, &c.Method,
		&tagsJSON, &c.Reasoning, &c.ClassifiedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("classification not found for email: %s", emailID)
	}
	if err != nil {
		return nil, err
	}

	// TODO: Parse tags JSON
	c.Tags = []string{}

	return c, nil
}

// ==================================
// Priority Operations
// ==================================

// Priority represents email priority scoring
type Priority struct {
	EmailID      string
	Score        int
	Factors      map[string]int
	Reasoning    string
	CalculatedAt time.Time
}

// SavePriority saves email priority
func (d *Database) SavePriority(p *Priority) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Convert factors to JSON (simplified)
	factorsJSON := "{}"

	query := `
		INSERT OR REPLACE INTO priorities (
			email_id, score, factors, reasoning
		) VALUES (?, ?, ?, ?)
	`

	_, err := d.db.Exec(query, p.EmailID, p.Score, factorsJSON, p.Reasoning)
	return err
}

// GetPriority retrieves priority for an email
func (d *Database) GetPriority(emailID string) (*Priority, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT email_id, score, factors, reasoning, calculated_at
		FROM priorities
		WHERE email_id = ?
	`

	p := &Priority{}
	var factorsJSON string

	err := d.db.QueryRow(query, emailID).Scan(
		&p.EmailID, &p.Score, &factorsJSON, &p.Reasoning, &p.CalculatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("priority not found for email: %s", emailID)
	}
	if err != nil {
		return nil, err
	}

	// TODO: Parse factors JSON
	p.Factors = make(map[string]int)

	return p, nil
}

// GetPriorityEmails retrieves emails sorted by priority
func (d *Database) GetPriorityEmails(accountID string, minScore, limit int) ([]*Email, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT e.id, e.account_id, e.message_id, e.thread_id, e.from_addr, e.to_addr,
			e.subject, e.body_snippet, e.received_at, e.read, e.starred, e.deleted,
			e.created_at, e.updated_at
		FROM emails e
		INNER JOIN priorities p ON e.id = p.email_id
		WHERE e.deleted = 0 AND p.score >= ?
	`

	args := []interface{}{minScore}

	if accountID != "" {
		query += " AND e.account_id = ?"
		args = append(args, accountID)
	}

	query += " ORDER BY p.score DESC, e.received_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emails := []*Email{}
	for rows.Next() {
		email := &Email{}
		err := rows.Scan(
			&email.ID, &email.AccountID, &email.MessageID, &email.ThreadID,
			&email.From, &email.To, &email.Subject, &email.BodySnippet,
			&email.ReceivedAt, &email.Read, &email.Starred, &email.Deleted,
			&email.CreatedAt, &email.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}

	return emails, rows.Err()
}

// ==================================
// Sender Analytics Operations
// ==================================

// SenderAnalytics represents sender statistics
type SenderAnalytics struct {
	EmailAddress    string
	AccountID       string
	TotalEmails     int
	ReadCount       int
	ReplyCount      int
	AvgResponseTime int  // minutes
	LastInteraction time.Time
	EngagementScore int
	IsVIP           bool
}

// UpdateSenderAnalytics updates or creates sender analytics
func (d *Database) UpdateSenderAnalytics(sa *SenderAnalytics) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `
		INSERT INTO sender_analytics (
			email_address, account_id, total_emails, read_count, reply_count,
			avg_response_time, last_interaction, engagement_score, is_vip
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(email_address) DO UPDATE SET
			total_emails = excluded.total_emails,
			read_count = excluded.read_count,
			reply_count = excluded.reply_count,
			avg_response_time = excluded.avg_response_time,
			last_interaction = excluded.last_interaction,
			engagement_score = excluded.engagement_score,
			is_vip = excluded.is_vip,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := d.db.Exec(query,
		sa.EmailAddress, sa.AccountID, sa.TotalEmails, sa.ReadCount,
		sa.ReplyCount, sa.AvgResponseTime, sa.LastInteraction,
		sa.EngagementScore, sa.IsVIP,
	)

	return err
}

// GetSenderAnalytics retrieves analytics for a sender
func (d *Database) GetSenderAnalytics(emailAddress string) (*SenderAnalytics, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT email_address, account_id, total_emails, read_count, reply_count,
			avg_response_time, last_interaction, engagement_score, is_vip
		FROM sender_analytics
		WHERE email_address = ?
	`

	sa := &SenderAnalytics{}
	err := d.db.QueryRow(query, emailAddress).Scan(
		&sa.EmailAddress, &sa.AccountID, &sa.TotalEmails, &sa.ReadCount,
		&sa.ReplyCount, &sa.AvgResponseTime, &sa.LastInteraction,
		&sa.EngagementScore, &sa.IsVIP,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("sender analytics not found: %s", emailAddress)
	}
	if err != nil {
		return nil, err
	}

	return sa, nil
}

// ==================================
// Utility Functions
// ==================================

// Vacuum optimizes the database
func (d *Database) Vacuum() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec("VACUUM")
	return err
}

// Stats returns database statistics
func (d *Database) Stats() (map[string]int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make(map[string]int)

	// Count emails
	err := d.db.QueryRow("SELECT COUNT(*) FROM emails WHERE deleted = 0").Scan(&stats["total_emails"])
	if err != nil {
		return nil, err
	}

	// Count unread
	err = d.db.QueryRow("SELECT COUNT(*) FROM emails WHERE deleted = 0 AND read = 0").Scan(&stats["unread_emails"])
	if err != nil {
		return nil, err
	}

	// Count high priority
	err = d.db.QueryRow("SELECT COUNT(*) FROM priorities WHERE score >= 70").Scan(&stats["high_priority"])
	if err != nil {
		return nil, err
	}

	return stats, nil
}
