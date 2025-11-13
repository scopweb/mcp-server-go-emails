# 🏗️ Arquitectura Técnica - MCP Email Server

## 📋 Tabla de Contenidos

1. [Visión General](#visión-general)
2. [Arquitectura de Alto Nivel](#arquitectura-de-alto-nivel)
3. [Componentes del Sistema](#componentes-del-sistema)
4. [Flujo de Datos](#flujo-de-datos)
5. [Base de Datos](#base-de-datos)
6. [APIs e Interfaces](#apis-e-interfaces)
7. [Seguridad](#seguridad)
8. [Performance y Escalabilidad](#performance-y-escalabilidad)
9. [Deployment](#deployment)

---

## Visión General

### Principios de Diseño

1. **Modularidad**: Componentes desacoplados con interfaces claras
2. **Escalabilidad**: Diseño que soporta crecimiento de cuentas y volumen
3. **Performance**: Respuestas <500ms para operaciones comunes
4. **Seguridad**: Encriptación end-to-end, zero-trust architecture
5. **Observabilidad**: Logging estructurado, métricas, tracing
6. **Extensibilidad**: Sistema de plugins para funcionalidades custom

### Stack Tecnológico

```
┌─────────────────────────────────────────────────┐
│                   Frontend                       │
│  (Claude Desktop, API Clients, Web Dashboard)   │
└───────────────────┬─────────────────────────────┘
                    │
                    │ MCP Protocol / REST API
                    │
┌───────────────────▼─────────────────────────────┐
│              MCP Server (Go)                     │
│  ┌──────────┬──────────┬──────────┬──────────┐ │
│  │   AI     │ Scheduler│  Search  │   API    │ │
│  │ Engine   │ Service  │  Engine  │ Gateway  │ │
│  └──────────┴──────────┴──────────┴──────────┘ │
└───────────────────┬─────────────────────────────┘
                    │
        ┌───────────┼───────────┐
        │           │           │
        ▼           ▼           ▼
   ┌────────┐  ┌────────┐  ┌────────┐
   │ SQLite │  │ Claude │  │ Email  │
   │  DB    │  │  API   │  │Servers │
   └────────┘  └────────┘  └────────┘
```

---

## Arquitectura de Alto Nivel

### Capas del Sistema

```
┌──────────────────────────────────────────────────────────┐
│                    Presentation Layer                     │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐        │
│  │MCP Protocol│  │ REST API   │  │  Web UI    │        │
│  └────────────┘  └────────────┘  └────────────┘        │
└─────────────────────────┬────────────────────────────────┘
                          │
┌─────────────────────────▼────────────────────────────────┐
│                     Business Logic Layer                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │Classifier│ │Priority  │ │Summarizer│ │Scheduler │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │  Search  │ │Analytics │ │Templates │ │  Notifs  │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘  │
└─────────────────────────┬────────────────────────────────┘
                          │
┌─────────────────────────▼────────────────────────────────┐
│                      Data Access Layer                    │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │Email Repo│ │Config    │ │Analytics │ │  Cache   │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘  │
└─────────────────────────┬────────────────────────────────┘
                          │
┌─────────────────────────▼────────────────────────────────┐
│                    Infrastructure Layer                   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │ SQLite   │ │  IMAP    │ │  SMTP    │ │ Claude   │  │
│  │   DB     │ │ Client   │ │ Client   │ │   API    │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘  │
└──────────────────────────────────────────────────────────┘
```

---

## Componentes del Sistema

### 1. Server Layer (`server/`)

**Responsabilidades:**
- Implementar protocolo MCP (JSON-RPC 2.0)
- Gestionar lifecycle de herramientas MCP
- Routing de requests a handlers
- Logging y error handling

**Componentes Clave:**

```go
// server/server.go
type MCPServer struct {
    config      *config.Config
    tools       map[string]Tool
    handlers    map[string]Handler
    logger      *zap.Logger
    middleware  []Middleware
}

// Inicialización
func NewMCPServer(cfg *config.Config) *MCPServer {
    server := &MCPServer{
        config:   cfg,
        tools:    make(map[string]Tool),
        handlers: make(map[string]Handler),
        logger:   initLogger(cfg),
    }

    server.registerTools()
    server.registerHandlers()
    return server
}

// Métodos principales
func (s *MCPServer) Start() error
func (s *MCPServer) HandleRequest(req MCPRequest) MCPResponse
func (s *MCPServer) RegisterTool(tool Tool) error
```

**Herramientas MCP (MVP):**

1. `classify_email` - Clasificar email por categoría
2. `priority_inbox` - Obtener emails prioritarios
3. `smart_filter` - Filtrar por múltiples criterios
4. `smart_summary` - Generar resumen inteligente
5. `email_analytics` - Métricas y estadísticas
6. `configure_rules` - Gestionar reglas de prioridad
7. `search_emails` - Búsqueda avanzada (V1.0)
8. `schedule_task` - Programar tareas (V1.0)

---

### 2. AI Engine (`ai/`)

**Responsabilidades:**
- Clasificación de emails
- Scoring de prioridades
- Generación de resúmenes
- Machine learning (V2.0)

#### 2.1 Clasificador (`ai/classifier.go`)

```go
type Classifier struct {
    rulesEngine *RulesEngine
    llmClient   *anthropic.Client
    cache       *Cache
    config      *ClassifierConfig
}

type EmailClassification struct {
    EmailID    string
    Category   string      // "work", "personal", "promotions", etc.
    Confidence float64     // 0.0 - 1.0
    Method     string      // "rules" | "ai" | "hybrid"
    Tags       []string
    Reasoning  string
    Timestamp  time.Time
}

type ClassifierConfig struct {
    UseAI           bool
    FallbackToRules bool
    ConfidenceMin   float64
    CacheEnabled    bool
    CacheTTL        time.Duration
}

// Métodos principales
func (c *Classifier) Classify(email *Email) (*EmailClassification, error)
func (c *Classifier) ClassifyBatch(emails []*Email) ([]*EmailClassification, error)
func (c *Classifier) LearnFromFeedback(emailID string, correctCategory string) error
```

**Algoritmo de Clasificación:**

```
1. Verificar cache
   └─> Si existe: return cached result

2. Aplicar reglas básicas
   ├─> Dominio del remitente → Categoría
   ├─> Keywords en subject → Categoría
   └─> Patrones de contenido → Categoría

3. Si confidence < threshold && AI enabled:
   ├─> Extraer features del email (subject, snippet, sender)
   ├─> Llamar a Claude API
   └─> Combinar resultado AI + rules (weighted average)

4. Guardar en cache
5. Return clasificación final
```

**Ejemplo de Reglas:**

```json
{
  "rules": [
    {
      "name": "work_domain",
      "condition": {
        "field": "from",
        "operator": "contains",
        "value": "@company.com"
      },
      "action": {
        "category": "work",
        "confidence": 0.95,
        "priority_boost": 20
      }
    },
    {
      "name": "invoice_detection",
      "condition": {
        "field": "subject",
        "operator": "regex",
        "value": "(?i)(invoice|factura|payment|pago)"
      },
      "action": {
        "category": "invoice",
        "confidence": 0.90,
        "priority_boost": 50,
        "tags": ["financial", "action_required"]
      }
    }
  ]
}
```

#### 2.2 Sistema de Prioridades (`ai/priority.go`)

```go
type PriorityEngine struct {
    config     *PriorityConfig
    rulesRepo  *RulesRepository
    analytics  *Analytics
    learner    *Learner
}

type PriorityScore struct {
    EmailID        string
    Score          int                // 0-100
    Factors        map[string]int     // Factor name -> contribution
    ReasoningChain []string           // Explicación del scoring
    Timestamp      time.Time
}

type PriorityFactors struct {
    SenderScore      int  // 0-30
    KeywordScore     int  // 0-20
    TemporalScore    int  // 0-15
    CategoryScore    int  // 0-15
    EngagementScore  int  // 0-10 (basado en histórico)
    ThreadScore      int  // 0-10
}

// Métodos principales
func (pe *PriorityEngine) CalculatePriority(email *Email) (*PriorityScore, error)
func (pe *PriorityEngine) UpdateRules(rules *PriorityRules) error
func (pe *PriorityEngine) GetTopPriority(accountID string, limit int) ([]*Email, error)
```

**Algoritmo de Scoring:**

```go
// Pseudo-código del algoritmo de prioridades

func calculatePriority(email Email, config PriorityConfig) int {
    score := 0
    reasoning := []string{}

    // Factor 1: Sender (0-30 puntos)
    if email.From in config.VIPSenders {
        score += 30
        reasoning = append(reasoning, "VIP sender (+30)")
    } else if email.FromDomain in config.ImportantDomains {
        score += 20
        reasoning = append(reasoning, "Important domain (+20)")
    }

    // Factor 2: Keywords urgentes (0-20 puntos)
    urgentKeywords := []string{"urgent", "asap", "deadline", "critical"}
    for _, kw := range urgentKeywords {
        if contains(email.Subject, kw) || contains(email.Body, kw) {
            score += 20
            reasoning = append(reasoning, fmt.Sprintf("Urgent keyword: %s (+20)", kw))
            break
        }
    }

    // Factor 3: Temporal (0-15 puntos)
    age := time.Since(email.ReceivedAt)
    if age < 1*time.Hour {
        score += 15
        reasoning = append(reasoning, "Very recent (+15)")
    } else if age < 6*time.Hour {
        score += 10
        reasoning = append(reasoning, "Recent (+10)")
    } else if age < 24*time.Hour {
        score += 5
        reasoning = append(reasoning, "Today (+5)")
    }

    // Factor 4: Categoría (0-15 puntos)
    categoryBoosts := config.CategoryPriority
    if boost, ok := categoryBoosts[email.Category]; ok {
        score += boost
        reasoning = append(reasoning, fmt.Sprintf("Category: %s (+%d)", email.Category, boost))
    }

    // Factor 5: Engagement histórico (0-10 puntos)
    if engagement := getEngagementScore(email.From); engagement > 0 {
        score += engagement
        reasoning = append(reasoning, fmt.Sprintf("Engagement history (+%d)", engagement))
    }

    // Factor 6: Thread (0-10 puntos)
    if email.InReplyTo != "" {
        threadPriority := getThreadPriority(email.ThreadID)
        score += min(threadPriority, 10)
        reasoning = append(reasoning, "Part of active thread (+10)")
    }

    // Normalizar a 0-100
    score = min(score, 100)

    return PriorityScore{
        Score:          score,
        ReasoningChain: reasoning,
    }
}
```

#### 2.3 Generador de Resúmenes (`ai/summarizer.go`)

```go
type Summarizer struct {
    llmClient  *anthropic.Client
    templates  *TemplateEngine
    analytics  *Analytics
}

type SummaryRequest struct {
    Accounts   []string
    Period     string      // "today" | "week" | "month" | "custom"
    DateFrom   time.Time
    DateTo     time.Time
    Categories []string
    MaxEmails  int
    Style      string      // "brief" | "detailed" | "executive"
    Sections   []string    // ["stats", "high_priority", "categories"]
}

type Summary struct {
    Title          string
    GeneratedAt    time.Time
    Period         string
    Statistics     *SummaryStats
    HighPriority   []*EmailDigest
    ByCategory     map[string][]*EmailDigest
    TopSenders     []SenderSummary
    Recommendations []string
}

// Métodos principales
func (s *Summarizer) GenerateSummary(req SummaryRequest) (*Summary, error)
func (s *Summarizer) GenerateEmailDigest(email *Email) (*EmailDigest, error)
```

**Ejemplo de Prompt para Claude:**

```go
const summaryPromptTemplate = `Generate a concise email summary for the following emails.

Context:
- Period: {{.Period}}
- Total emails: {{.TotalEmails}}
- Accounts: {{.Accounts}}
- Style: {{.Style}}

High Priority Emails ({{.HighPriorityCount}}):
{{range .HighPriorityEmails}}
- From: {{.From}}
  Subject: {{.Subject}}
  Received: {{.ReceivedAt}}
  Priority: {{.PriorityScore}}/100
  Snippet: {{.BodySnippet}}
{{end}}

Other Notable Emails:
{{range .OtherEmails}}
- [{{.Category}}] {{.From}}: {{.Subject}}
{{end}}

Please provide:
1. Executive summary (2-3 sentences)
2. Action items (emails requiring immediate attention)
3. Notable patterns or trends
4. Key statistics

Format: Markdown`
```

---

### 3. Storage Layer (`storage/`)

#### 3.1 Esquema de Base de Datos

```sql
-- storage/schema.sql

-- Tabla principal de emails (cache)
CREATE TABLE emails (
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
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_account_received (account_id, received_at DESC),
    INDEX idx_thread (thread_id),
    INDEX idx_from (from_addr)
);

-- Clasificaciones
CREATE TABLE classifications (
    email_id TEXT PRIMARY KEY,
    category TEXT NOT NULL,
    confidence REAL NOT NULL,
    method TEXT NOT NULL,
    tags TEXT, -- JSON array
    reasoning TEXT,
    classified_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE,
    INDEX idx_category (category)
);

-- Prioridades
CREATE TABLE priorities (
    email_id TEXT PRIMARY KEY,
    score INTEGER NOT NULL,
    factors TEXT NOT NULL, -- JSON object
    reasoning TEXT,
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE,
    INDEX idx_score (score DESC)
);

-- Análisis de remitentes
CREATE TABLE sender_analytics (
    email_address TEXT PRIMARY KEY,
    account_id TEXT,
    total_emails INTEGER DEFAULT 0,
    read_count INTEGER DEFAULT 0,
    reply_count INTEGER DEFAULT 0,
    avg_response_time INTEGER, -- en minutos
    last_interaction TIMESTAMP,
    engagement_score INTEGER DEFAULT 0,
    is_vip BOOLEAN DEFAULT 0,

    INDEX idx_engagement (engagement_score DESC)
);

-- Búsqueda full-text
CREATE VIRTUAL TABLE emails_fts USING fts5(
    email_id UNINDEXED,
    subject,
    body_snippet,
    from_addr,
    tokenize='porter unicode61'
);

-- Triggers para mantener FTS sincronizado
CREATE TRIGGER emails_ai AFTER INSERT ON emails BEGIN
    INSERT INTO emails_fts(email_id, subject, body_snippet, from_addr)
    VALUES (NEW.id, NEW.subject, NEW.body_snippet, NEW.from_addr);
END;

CREATE TRIGGER emails_ad AFTER DELETE ON emails BEGIN
    DELETE FROM emails_fts WHERE email_id = OLD.id;
END;

-- User actions (para machine learning)
CREATE TABLE user_actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email_id TEXT NOT NULL,
    action TEXT NOT NULL, -- "read" | "delete" | "star" | "reply" | "archive"
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    context TEXT, -- JSON

    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE,
    INDEX idx_email_action (email_id, action)
);

-- Tareas programadas
CREATE TABLE scheduled_tasks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    cron_expr TEXT NOT NULL,
    action TEXT NOT NULL,
    config TEXT, -- JSON
    enabled BOOLEAN DEFAULT 1,
    last_run TIMESTAMP,
    next_run TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Configuraciones (key-value store)
CREATE TABLE configs (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### 3.2 Repository Pattern

```go
// storage/email_repository.go

type EmailRepository struct {
    db     *sql.DB
    cache  *bbolt.DB
    logger *zap.Logger
}

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

    // Relaciones cargadas lazy
    Classification *EmailClassification
    Priority       *PriorityScore
}

// Métodos principales
func (r *EmailRepository) Create(email *Email) error
func (r *EmailRepository) Get(id string) (*Email, error)
func (r *EmailRepository) List(filter EmailFilter) ([]*Email, error)
func (r *EmailRepository) Update(email *Email) error
func (r *EmailRepository) Delete(id string) error

// Queries especializadas
func (r *EmailRepository) GetByPriority(accountID string, minScore int, limit int) ([]*Email, error)
func (r *EmailRepository) GetByCategory(accountID, category string, limit int) ([]*Email, error)
func (r *EmailRepository) Search(query SearchQuery) ([]*Email, error)
```

---

### 4. Scheduler Service (`scheduler/`)

```go
// scheduler/cron.go

type Scheduler struct {
    cron      *cron.Cron
    tasks     map[string]*Task
    notifier  *Notifier
    logger    *zap.Logger
}

type Task struct {
    ID       string
    Name     string
    Schedule string // cron expression
    Handler  TaskHandler
    Config   map[string]interface{}
    Enabled  bool
}

type TaskHandler func(ctx context.Context, config map[string]interface{}) error

// Tareas predefinidas
func (s *Scheduler) RegisterDefaultTasks() {
    s.RegisterTask("daily_summary", "0 9 * * *", s.generateDailySummary)
    s.RegisterTask("priority_scan", "*/15 * * * *", s.scanPriorityEmails)
    s.RegisterTask("cache_cleanup", "0 2 * * *", s.cleanupOldCache)
    s.RegisterTask("analytics_update", "0 */6 * * *", s.updateAnalytics)
}

// Sistema de notificaciones
type Notifier struct {
    webhooks  []WebhookConfig
    email     *EmailSender
    logger    *zap.Logger
}

type Notification struct {
    Type     string // "webhook" | "email" | "desktop"
    Priority string // "low" | "medium" | "high" | "critical"
    Title    string
    Message  string
    Data     interface{}
}

func (n *Notifier) Send(notification *Notification) error
```

---

### 5. Search Engine (`search/`)

```go
// search/engine.go

type SearchEngine struct {
    db        *sql.DB
    ftsIndex  *FTSIndex
    cache     *Cache
}

type SearchQuery struct {
    Text        string
    Accounts    []string
    Categories  []string
    Senders     []string
    MinPriority int
    DateFrom    time.Time
    DateTo      time.Time
    Read        *bool
    Starred     *bool
    Limit       int
    Offset      int
}

type SearchResult struct {
    Emails      []*Email
    TotalCount  int
    SearchTime  time.Duration
    Suggestions []string
}

// Búsqueda full-text con SQLite FTS5
func (se *SearchEngine) Search(query SearchQuery) (*SearchResult, error) {
    startTime := time.Now()

    // Construir query SQL
    sql := `
        SELECT e.*,
               snippet(emails_fts, 1, '<mark>', '</mark>', '...', 64) as highlight
        FROM emails e
        JOIN emails_fts ON emails_fts.email_id = e.id
        WHERE emails_fts MATCH ?
    `

    // Agregar filtros
    if len(query.Accounts) > 0 {
        sql += " AND e.account_id IN (?)"
    }

    if query.MinPriority > 0 {
        sql += ` AND e.id IN (
            SELECT email_id FROM priorities WHERE score >= ?
        )`
    }

    // ... más filtros

    sql += " ORDER BY e.received_at DESC LIMIT ? OFFSET ?"

    // Ejecutar query
    rows, err := se.db.Query(sql, params...)
    // ... procesar resultados

    return &SearchResult{
        Emails:     emails,
        TotalCount: count,
        SearchTime: time.Since(startTime),
    }, nil
}
```

---

## Flujo de Datos

### Flujo 1: Clasificación de Email

```
┌──────────┐
│  User    │
│ Request  │
└────┬─────┘
     │
     │ classify_email
     ▼
┌────────────────┐
│  MCP Handler   │
└────┬───────────┘
     │
     │ Get Email
     ▼
┌────────────────┐      Cache Miss
│ Email Repo     ├──────────────────┐
└────┬───────────┘                  │
     │                               │
     │ Cache Hit                     ▼
     ▼                          ┌──────────┐
┌────────────────┐              │  IMAP    │
│ Classifier     │              │  Fetch   │
└────┬───────────┘              └────┬─────┘
     │                               │
     │ Apply Rules                   │ Store
     ▼                               ▼
┌────────────────┐              ┌──────────┐
│ Rules Engine   │              │   DB     │
└────┬───────────┘              └──────────┘
     │
     │ confidence < threshold?
     ▼
┌────────────────┐
│  Claude API    │ (opcional)
└────┬───────────┘
     │
     │ Combine results
     ▼
┌────────────────┐
│  Save to DB    │
└────┬───────────┘
     │
     ▼
┌────────────────┐
│  Return to     │
│     User       │
└────────────────┘
```

### Flujo 2: Resumen Inteligente

```
User Request "smart_summary"
      │
      ▼
Get emails from DB (filtered by date, account)
      │
      ▼
For each email:
  ├─> Get classification
  ├─> Get priority score
  └─> Get sender analytics
      │
      ▼
Aggregate statistics:
  ├─> Total emails
  ├─> By category
  ├─> High priority count
  └─> Top senders
      │
      ▼
Prepare prompt for Claude API
      │
      ▼
Call Claude API with email data
      │
      ▼
Parse LLM response
      │
      ▼
Format as markdown summary
      │
      ▼
Cache summary (1 hour TTL)
      │
      ▼
Return to user
```

---

## Seguridad

### 1. Encriptación de Credenciales

```go
// config/crypto.go

type SecureConfigManager struct {
    masterKey []byte
    cipher    cipher.AEAD
}

// Derivar master key de passphrase del usuario
func DeriveKey(passphrase string, salt []byte) []byte {
    return argon2.IDKey(
        []byte(passphrase),
        salt,
        1,          // time cost
        64*1024,    // memory cost (64 MB)
        4,          // parallelism
        32,         // key length
    )
}

// Encriptar passwords
func (scm *SecureConfigManager) Encrypt(plaintext []byte) ([]byte, error) {
    nonce := make([]byte, scm.cipher.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    return scm.cipher.Seal(nonce, nonce, plaintext, nil), nil
}

// Formato en disco:
// email_config.encrypted.json
{
  "version": "1",
  "kdf": "argon2id",
  "salt": "base64_encoded_salt",
  "accounts": {
    "personal": {
      "imap_host": "imap.gmail.com",
      "username": "user@gmail.com",
      "password": "encrypted:base64_encoded_encrypted_password"
    }
  }
}
```

### 2. Rate Limiting

```go
// utils/ratelimiter.go

type RateLimiter struct {
    limits map[string]*rate.Limiter
    mu     sync.RWMutex
}

// Configuración
var rateLimits = map[string]RateLimit{
    "imap_connection":  {Requests: 5, Per: time.Minute},
    "claude_api":       {Requests: 50, Per: time.Minute},
    "classification":   {Requests: 100, Per: time.Minute},
    "priority_scan":    {Requests: 200, Per: time.Minute},
}

func (rl *RateLimiter) Allow(key string) bool {
    rl.mu.RLock()
    limiter, exists := rl.limits[key]
    rl.mu.RUnlock()

    if !exists {
        rl.mu.Lock()
        limiter = rate.NewLimiter(rate.Every(time.Minute/50), 50)
        rl.limits[key] = limiter
        rl.mu.Unlock()
    }

    return limiter.Allow()
}
```

### 3. Input Validation

```go
// utils/validator.go

type EmailValidator struct {
    v *validator.Validate
}

type ClassifyEmailRequest struct {
    EmailID   string `validate:"required,uuid"`
    AccountID string `validate:"required,alphanum"`
    UseAI     bool   `validate:""`
}

func (ev *EmailValidator) Validate(req interface{}) error {
    if err := ev.v.Struct(req); err != nil {
        return fmt.Errorf("validation error: %w", err)
    }
    return nil
}

// Sanitización de queries SQL
func SanitizeSearchQuery(query string) string {
    // Remover caracteres peligrosos para FTS5
    query = strings.ReplaceAll(query, "\"", "\\\"")
    query = strings.ReplaceAll(query, "'", "\\'")

    // Limitar longitud
    if len(query) > 500 {
        query = query[:500]
    }

    return query
}
```

---

## Performance y Escalabilidad

### 1. Caching Strategy

```go
// storage/cache.go

type CacheLayer struct {
    memory  *ristretto.Cache  // In-memory cache (hot data)
    disk    *bbolt.DB          // On-disk cache (warm data)
    ttl     time.Duration
}

// Estrategia de caching
type CacheStrategy struct {
    // L1: In-memory cache (ristretto) - 128MB
    // - Clasificaciones recientes (1 hora)
    // - Prioridades recientes (30 min)
    // - Resúmenes (1 hora)

    // L2: Disk cache (bbolt) - 1GB
    // - Clasificaciones históricas (7 días)
    // - Analytics agregados (1 día)
    // - Resúmenes (24 horas)

    // L3: SQLite database - illimitado
    // - Todos los datos persistentes
}

// Ejemplo de uso
func (c *CacheLayer) GetOrCompute(
    key string,
    compute func() (interface{}, error),
) (interface{}, error) {
    // Intentar L1
    if val, found := c.memory.Get(key); found {
        return val, nil
    }

    // Intentar L2
    var diskVal []byte
    err := c.disk.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket([]byte("cache"))
        diskVal = b.Get([]byte(key))
        return nil
    })

    if err == nil && diskVal != nil {
        // Deserializar y guardar en L1
        val := deserialize(diskVal)
        c.memory.Set(key, val, 1)
        return val, nil
    }

    // Computar y guardar en ambos niveles
    val, err := compute()
    if err != nil {
        return nil, err
    }

    c.memory.Set(key, val, 1)
    c.saveToDisk(key, val)

    return val, nil
}
```

### 2. Batch Processing

```go
// ai/batch_processor.go

type BatchProcessor struct {
    batchSize int
    timeout   time.Duration
    workers   int
}

// Clasificación en batch
func (bp *BatchProcessor) ClassifyBatch(emails []*Email) ([]*EmailClassification, error) {
    results := make([]*EmailClassification, len(emails))
    errors := make([]error, len(emails))

    // Dividir en chunks
    chunks := chunkEmails(emails, bp.batchSize)

    // Procesar en paralelo con worker pool
    var wg sync.WaitGroup
    sem := make(chan struct{}, bp.workers)

    for i, chunk := range chunks {
        wg.Add(1)
        go func(idx int, emails []*Email) {
            defer wg.Done()
            sem <- struct{}{}        // Acquire
            defer func() { <-sem }() // Release

            // Procesar chunk
            for j, email := range emails {
                classification, err := bp.classifier.Classify(email)
                globalIdx := idx*bp.batchSize + j
                results[globalIdx] = classification
                errors[globalIdx] = err
            }
        }(i, chunk)
    }

    wg.Wait()

    return results, combineErrors(errors)
}
```

### 3. Database Optimization

```sql
-- Índices compuestos para queries frecuentes
CREATE INDEX idx_priority_emails ON emails(account_id, received_at DESC)
WHERE deleted = 0;

CREATE INDEX idx_unread_priority ON emails(account_id, read)
INCLUDE (id, subject, from_addr, received_at)
WHERE deleted = 0 AND read = 0;

-- Particionamiento lógico por fecha (cleanup automático)
CREATE TRIGGER auto_cleanup AFTER INSERT ON emails
BEGIN
    DELETE FROM emails
    WHERE received_at < datetime('now', '-30 days')
    AND starred = 0;
END;

-- Estadísticas para query optimizer
ANALYZE emails;
ANALYZE classifications;
ANALYZE priorities;
```

### 4. Métricas y Monitoring

```go
// utils/metrics.go

type Metrics struct {
    registry *prometheus.Registry

    // Counters
    classificationsTotal  prometheus.Counter
    apiCallsTotal         prometheus.Counter
    errorsTotal           prometheus.Counter

    // Histograms
    classificationDuration prometheus.Histogram
    priorityDuration       prometheus.Histogram
    searchDuration         prometheus.Histogram

    // Gauges
    cacheSize             prometheus.Gauge
    activeConnections     prometheus.Gauge
}

func (m *Metrics) RecordClassification(duration time.Duration, method string) {
    m.classificationsTotal.Inc()
    m.classificationDuration.Observe(duration.Seconds())
}

// Exponemos métricas en /metrics endpoint
func (m *Metrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}
```

---

## Deployment

### Opciones de Deployment

#### 1. Desarrollo Local

```bash
# Compilar
go build -o email-mcp-server main.go

# Ejecutar
./email-mcp-server --config config/email_config.json
```

#### 2. Docker

```dockerfile
# Dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o email-mcp-server main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/email-mcp-server .
COPY --from=builder /app/config ./config

EXPOSE 8080
CMD ["./email-mcp-server"]
```

```yaml
# docker-compose.yml
version: '3.8'

services:
  email-server:
    build: .
    volumes:
      - ./config:/root/config
      - ./data:/root/data
    environment:
      - LOG_LEVEL=info
      - CLAUDE_API_KEY=${CLAUDE_API_KEY}
    restart: unless-stopped
```

#### 3. Systemd Service

```ini
# /etc/systemd/system/email-mcp-server.service
[Unit]
Description=MCP Email Server
After=network.target

[Service]
Type=simple
User=emailserver
WorkingDirectory=/opt/email-mcp-server
ExecStart=/opt/email-mcp-server/email-mcp-server
Restart=on-failure
RestartSec=10

Environment="CLAUDE_API_KEY=your-key-here"

[Install]
WantedBy=multi-user.target
```

---

## Consideraciones Adicionales

### Testing Strategy

```
├── test/
│   ├── unit/           # Tests unitarios (>80% cobertura)
│   │   ├── classifier_test.go
│   │   ├── priority_test.go
│   │   └── summarizer_test.go
│   │
│   ├── integration/    # Tests de integración
│   │   ├── email_flow_test.go
│   │   ├── api_test.go
│   │   └── scheduler_test.go
│   │
│   ├── e2e/           # Tests end-to-end
│   │   └── full_workflow_test.go
│   │
│   └── fixtures/      # Datos de prueba
│       ├── sample_emails.json
│       └── mock_responses.json
```

### Logging Strategy

```go
// utils/logger.go

logger, _ := zap.NewProduction()
defer logger.Sync()

// Structured logging
logger.Info("email classified",
    zap.String("email_id", emailID),
    zap.String("category", category),
    zap.Float64("confidence", confidence),
    zap.Duration("duration", duration),
)

// Niveles de log por componente
loggerConfig := zap.Config{
    Level: zap.NewAtomicLevelAt(zap.InfoLevel),
    OutputPaths: []string{
        "stdout",
        "/var/log/email-mcp-server/app.log",
    },
    ErrorOutputPaths: []string{
        "stderr",
        "/var/log/email-mcp-server/error.log",
    },
}
```

---

**Última actualización**: 2025-11-13
**Versión**: 1.0
**Autor**: MCP Email Server Team
