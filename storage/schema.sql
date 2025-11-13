-- Email Server Database Schema
-- Version: 1.0
-- Database: SQLite 3

-- ============================================
-- Tabla principal de emails (cache)
-- ============================================
CREATE TABLE IF NOT EXISTS emails (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL,
    message_id TEXT UNIQUE,
    thread_id TEXT,
    from_addr TEXT NOT NULL,
    to_addr TEXT,
    subject TEXT,
    body_snippet TEXT,
    received_at TIMESTAMP NOT NULL,
    read BOOLEAN DEFAULT 0,
    starred BOOLEAN DEFAULT 0,
    deleted BOOLEAN DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_account_received ON emails(account_id, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_thread ON emails(thread_id);
CREATE INDEX IF NOT EXISTS idx_from ON emails(from_addr);
CREATE INDEX IF NOT EXISTS idx_deleted ON emails(deleted) WHERE deleted = 0;

-- ============================================
-- Clasificaciones
-- ============================================
CREATE TABLE IF NOT EXISTS classifications (
    email_id TEXT PRIMARY KEY,
    category TEXT NOT NULL,
    confidence REAL NOT NULL CHECK(confidence >= 0 AND confidence <= 1),
    method TEXT NOT NULL CHECK(method IN ('rules', 'ai', 'hybrid')),
    tags TEXT, -- JSON array
    reasoning TEXT,
    classified_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_category ON classifications(category);
CREATE INDEX IF NOT EXISTS idx_method ON classifications(method);

-- ============================================
-- Prioridades
-- ============================================
CREATE TABLE IF NOT EXISTS priorities (
    email_id TEXT PRIMARY KEY,
    score INTEGER NOT NULL CHECK(score >= 0 AND score <= 100),
    factors TEXT NOT NULL, -- JSON object
    reasoning TEXT,
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_score ON priorities(score DESC);
CREATE INDEX IF NOT EXISTS idx_score_calculated ON priorities(score DESC, calculated_at DESC);

-- ============================================
-- Análisis de remitentes
-- ============================================
CREATE TABLE IF NOT EXISTS sender_analytics (
    email_address TEXT PRIMARY KEY,
    account_id TEXT,
    total_emails INTEGER DEFAULT 0,
    read_count INTEGER DEFAULT 0,
    reply_count INTEGER DEFAULT 0,
    avg_response_time INTEGER, -- en minutos
    last_interaction TIMESTAMP,
    engagement_score INTEGER DEFAULT 0 CHECK(engagement_score >= 0 AND engagement_score <= 100),
    is_vip BOOLEAN DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_engagement ON sender_analytics(engagement_score DESC);
CREATE INDEX IF NOT EXISTS idx_vip ON sender_analytics(is_vip) WHERE is_vip = 1;

-- ============================================
-- Búsqueda full-text (FTS5)
-- ============================================
CREATE VIRTUAL TABLE IF NOT EXISTS emails_fts USING fts5(
    email_id UNINDEXED,
    subject,
    body_snippet,
    from_addr,
    tokenize='porter unicode61'
);

-- ============================================
-- Triggers para mantener FTS sincronizado
-- ============================================
CREATE TRIGGER IF NOT EXISTS emails_fts_insert AFTER INSERT ON emails BEGIN
    INSERT INTO emails_fts(email_id, subject, body_snippet, from_addr)
    VALUES (NEW.id, NEW.subject, NEW.body_snippet, NEW.from_addr);
END;

CREATE TRIGGER IF NOT EXISTS emails_fts_delete AFTER DELETE ON emails BEGIN
    DELETE FROM emails_fts WHERE email_id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS emails_fts_update AFTER UPDATE ON emails BEGIN
    DELETE FROM emails_fts WHERE email_id = OLD.id;
    INSERT INTO emails_fts(email_id, subject, body_snippet, from_addr)
    VALUES (NEW.id, NEW.subject, NEW.body_snippet, NEW.from_addr);
END;

-- ============================================
-- User actions (para machine learning)
-- ============================================
CREATE TABLE IF NOT EXISTS user_actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email_id TEXT NOT NULL,
    action TEXT NOT NULL CHECK(action IN ('read', 'delete', 'star', 'reply', 'archive', 'mark_spam')),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    context TEXT, -- JSON
    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_email_action ON user_actions(email_id, action);
CREATE INDEX IF NOT EXISTS idx_action_timestamp ON user_actions(action, timestamp DESC);

-- ============================================
-- Tareas programadas
-- ============================================
CREATE TABLE IF NOT EXISTS scheduled_tasks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    cron_expr TEXT NOT NULL,
    action TEXT NOT NULL,
    config TEXT, -- JSON
    enabled BOOLEAN DEFAULT 1,
    last_run TIMESTAMP,
    next_run TIMESTAMP,
    last_status TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_enabled_next_run ON scheduled_tasks(enabled, next_run) WHERE enabled = 1;

-- ============================================
-- Configuraciones (key-value store)
-- ============================================
CREATE TABLE IF NOT EXISTS configs (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- Cache de resúmenes
-- ============================================
CREATE TABLE IF NOT EXISTS summaries_cache (
    id TEXT PRIMARY KEY,
    accounts TEXT NOT NULL, -- JSON array
    period TEXT NOT NULL,
    date_from TIMESTAMP,
    date_to TIMESTAMP,
    summary_data TEXT NOT NULL, -- JSON
    generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_expires ON summaries_cache(expires_at);

-- ============================================
-- Trigger para cleanup automático de datos antiguos
-- ============================================
CREATE TRIGGER IF NOT EXISTS auto_cleanup_emails AFTER INSERT ON emails BEGIN
    DELETE FROM emails
    WHERE received_at < datetime('now', '-30 days')
    AND starred = 0
    AND deleted = 0;
END;

CREATE TRIGGER IF NOT EXISTS auto_cleanup_summaries AFTER INSERT ON summaries_cache BEGIN
    DELETE FROM summaries_cache
    WHERE expires_at < datetime('now');
END;

-- ============================================
-- Trigger para updated_at automático
-- ============================================
CREATE TRIGGER IF NOT EXISTS emails_updated_at AFTER UPDATE ON emails BEGIN
    UPDATE emails SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS sender_analytics_updated_at AFTER UPDATE ON sender_analytics BEGIN
    UPDATE sender_analytics SET updated_at = CURRENT_TIMESTAMP WHERE email_address = NEW.email_address;
END;

CREATE TRIGGER IF NOT EXISTS scheduled_tasks_updated_at AFTER UPDATE ON scheduled_tasks BEGIN
    UPDATE scheduled_tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- ============================================
-- Configuraciones iniciales
-- ============================================
INSERT OR IGNORE INTO configs (key, value, description) VALUES
('schema_version', '1.0', 'Database schema version'),
('created_at', datetime('now'), 'Database creation timestamp'),
('cache_retention_days', '30', 'Days to keep cached emails'),
('summary_cache_hours', '1', 'Hours to keep summary cache');

-- ============================================
-- Estadísticas y análisis
-- ============================================
-- Vista para emails prioritarios
CREATE VIEW IF NOT EXISTS priority_emails AS
SELECT
    e.*,
    c.category,
    c.confidence,
    p.score as priority_score,
    p.reasoning as priority_reasoning
FROM emails e
LEFT JOIN classifications c ON e.id = c.email_id
LEFT JOIN priorities p ON e.id = p.email_id
WHERE e.deleted = 0
ORDER BY p.score DESC, e.received_at DESC;

-- Vista para resumen por categoría
CREATE VIEW IF NOT EXISTS emails_by_category AS
SELECT
    e.account_id,
    c.category,
    COUNT(*) as count,
    SUM(CASE WHEN e.read = 0 THEN 1 ELSE 0 END) as unread_count,
    AVG(p.score) as avg_priority_score,
    MAX(e.received_at) as latest_email
FROM emails e
LEFT JOIN classifications c ON e.id = c.email_id
LEFT JOIN priorities p ON e.id = p.email_id
WHERE e.deleted = 0
GROUP BY e.account_id, c.category;

-- Vista para análisis de remitentes con emails
CREATE VIEW IF NOT EXISTS sender_email_stats AS
SELECT
    s.*,
    COUNT(e.id) as email_count,
    SUM(CASE WHEN e.read = 1 THEN 1 ELSE 0 END) as read_emails,
    AVG(p.score) as avg_priority
FROM sender_analytics s
LEFT JOIN emails e ON s.email_address = e.from_addr AND e.deleted = 0
LEFT JOIN priorities p ON e.id = p.email_id
GROUP BY s.email_address;
