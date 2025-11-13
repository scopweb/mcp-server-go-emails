# 📘 MCP Email Server - Usage Guide

## Table of Contents

1. [Quick Start](#quick-start)
2. [Intelligent Tools](#intelligent-tools)
3. [Configuration](#configuration)
4. [Integration Examples](#integration-examples)
5. [API Reference](#api-reference)

---

## Quick Start

### 1. Setup Configuration

Create your configuration files:

```bash
# Copy example configurations
cp config/priority_rules.example.json config/priority_rules.json
cp config/ai_config.example.json config/ai_config.json
cp email_config.example.json email_config.json

# Edit with your settings
nano config/priority_rules.json
```

### 2. Initialize Database

The database will be automatically created on first run at `./data/emails.db`.

### 3. Use with Claude Desktop

Add the new tools to your Claude Desktop configuration:

```json
{
  "mcpServers": {
    "email": {
      "command": "/path/to/email-mcp-server",
      "args": [],
      "env": {
        "DB_PATH": "./data/emails.db",
        "CONFIG_PATH": "./config/priority_rules.json"
      }
    }
  }
}
```

---

## Intelligent Tools

### 1. `classify_email` - Intelligent Email Classification

Automatically categorizes emails into: work, personal, promotions, invoice, newsletters, urgent.

**Parameters:**
- `from` (required): Sender email address
- `subject` (required): Email subject
- `body_snippet` (optional): First 500 chars of email body
- `email_id` (optional): Email ID for saving to database

**Example Usage:**

```
User: "Classify this email: from: newsletter@techcrunch.com, subject: Daily Tech News Digest"

Claude: Using classify_email tool...

Response:
📧 Email Classification Result

Category: newsletters
Confidence: 90%
Method: rules
Tags: [newsletter, bulk]

Reasoning: Newsletter keywords in subject

From: newsletter@techcrunch.com
Subject: Daily Tech News Digest
```

**Use Cases:**
- Auto-organize incoming mail
- Filter spam and promotions
- Identify important work emails
- Detect invoices automatically

---

### 2. `priority_inbox` - Smart Priority Inbox

Returns emails sorted by intelligent priority score (0-100).

**Parameters:**
- `account` (optional): Account ID
- `min_score` (optional): Minimum priority score (default: 70)
- `limit` (optional): Max emails to return (default: 20)

**Example Usage:**

```
User: "Show me my most important emails"

Claude: Using priority_inbox with min_score: 70...

Response:
🎯 Priority Inbox (score >= 70)

Found 5 high-priority emails:

1. 🔴 [work] Q4 Budget Review - URGENT
   From: ceo@company.com
   Score: 95/100
   Received: 2025-01-13 09:30
   Why: VIP sender (+30); Urgent keyword in subject: 'URGENT' (+20); Very recent: <1 hour (+15)

2. 🟠 [invoice] Invoice #12345 Due Tomorrow
   From: billing@vendor.com
   Score: 85/100
   Received: 2025-01-13 08:15
   Why: Category 'invoice' (+20); Action keyword: 'due' (+10); Recent: <6 hours (+10)

3. 🟠 [work] Project Deadline Update
   From: manager@company.com
   Score: 78/100
   Received: 2025-01-12 16:45
   Why: Important domain: company.com (+20); Action keyword: 'deadline' (+10)
```

**Priority Levels:**
- 🔴 90-100: CRITICAL (immediate attention required)
- 🟠 70-89: HIGH (important, address today)
- 🟡 40-69: MEDIUM (normal priority)
- 🟢 20-39: LOW (can wait)
- ⚪ 0-19: MINIMAL (low importance)

---

### 3. `smart_filter` - Advanced Email Filtering

Filter emails by multiple intelligent criteria.

**Parameters:**
- `account` (optional): Account ID
- `category` (optional): Filter by category
- `min_priority` (optional): Minimum priority score
- `unread_only` (optional): Show only unread
- `date_from` (optional): From date (YYYY-MM-DD)
- `date_to` (optional): To date (YYYY-MM-DD)
- `limit` (optional): Max results (default: 50)

**Example Usage:**

```
User: "Show me unread work emails from this week with high priority"

Claude: Using smart_filter...

Response:
🔍 Smart Filter Results

Filters applied:
  • Category: work
  • Min Priority: 70
  • Unread only: true
  • Date from: 2025-01-08

Found 8 emails:

1. [work] Team Meeting Agenda - Please Review
   From: manager@company.com | Priority: 82/100
   Date: 2025-01-12 14:30

2. [work] Code Review Request - Urgent
   From: dev@company.com | Priority: 75/100
   Date: 2025-01-11 10:15
```

---

### 4. `analyze_priority` - Priority Score Explainer

Get detailed breakdown of why an email has a specific priority score.

**Parameters:**
- `email_id` (optional): Email ID from database
- `from` (required if no email_id): Sender
- `subject` (required if no email_id): Subject
- `body_snippet` (optional): Body preview
- `received_at` (optional): When received (RFC3339)

**Example Usage:**

```
User: "Why does this email have high priority? from: ceo@company.com, subject: Board Meeting Tomorrow 9 AM"

Claude: Using analyze_priority...

Response:
Priority Score: 95/100

🔴 CRITICAL PRIORITY

Score Breakdown:
  • ✅ VIP sender (+30)
  • 🚨 Urgent keyword in subject: 'board meeting' (+20)
  • ⏰ Very recent: <1 hour (+15)
  • 📁 Category 'work' (+15)
  • 💬 High engagement: 95% read, 60% reply (+9)
  • 📝 No urgent keywords (+0)
```

**Score Factors:**
1. **Sender (0-30)**: VIP status, domain importance, engagement
2. **Keywords (0-20)**: Urgent terms detection
3. **Temporal (0-15)**: Email recency
4. **Category (0-15)**: Email type importance
5. **Engagement (0-10)**: Your interaction history
6. **Thread (0-10)**: Conversation context

---

## Configuration

### Priority Rules (`config/priority_rules.json`)

```json
{
  "priority_rules": {
    "vip_senders": [
      "boss@company.com",
      "ceo@company.com"
    ],
    "important_domains": [
      "company.com",
      "client.com"
    ],
    "urgent_keywords": [
      "urgent",
      "asap",
      "critical",
      "deadline",
      "important"
    ],
    "ignore_senders": [
      "noreply@*",
      "*@marketing.com"
    ],
    "category_priority": {
      "work": 15,
      "invoice": 20,
      "urgent": 25,
      "personal": 10,
      "promotions": -20,
      "newsletters": -30
    }
  },
  "classification_rules": {
    "work": {
      "description": "Work-related emails",
      "conditions": [
        {
          "field": "from",
          "operator": "domain_in",
          "values": ["company.com"]
        }
      ],
      "priority_boost": 20,
      "confidence": 0.95
    },
    "invoice": {
      "description": "Invoices and payments",
      "conditions": [
        {
          "field": "subject",
          "operator": "regex",
          "value": "(?i)(invoice|payment|receipt)"
        }
      ],
      "priority_boost": 50,
      "confidence": 0.90,
      "tags": ["financial", "action_required"]
    }
  }
}
```

### Customization Examples

**Add VIP Sender:**
```json
"vip_senders": [
  "boss@company.com",
  "important-client@example.com",
  "board@company.com"
]
```

**Add Custom Category:**
```json
"customers": {
  "description": "Customer emails",
  "conditions": [
    {
      "field": "from",
      "operator": "domain_in",
      "values": ["customer1.com", "customer2.com"]
    }
  ],
  "priority_boost": 30,
  "confidence": 0.95,
  "tags": ["customer", "high_priority"]
}
```

**Multi-language Keywords:**
```json
"urgent_keywords": [
  "urgent", "asap", "critical",
  "urgente", "importante",        // Spanish
  "urgent", "wichtig", "dringend"  // German
]
```

---

## Integration Examples

### Integrate with Existing MCP Server

Add to your `main.go` handleToolCall function:

```go
import (
    "email-mcp-server/server"
    "email-mcp-server/utils"
)

// Initialize intelligent server (once, at startup)
var intelligentServer *server.IntelligentEmailServer

func init() {
    var err error
    intelligentServer, err = server.NewIntelligentEmailServer(
        "./data/emails.db",
        "./config/priority_rules.json",
    )
    if err != nil {
        log.Fatal("Failed to initialize intelligent server:", err)
    }
}

// In handleToolCall function, add:
func (es *EmailServer) handleToolCall(params ToolCallParams) (interface{}, error) {
    switch params.Name {

    // ... existing cases ...

    case "classify_email":
        result, err := intelligentServer.HandleClassifyEmail(params.Arguments)
        if err != nil {
            return nil, err
        }
        return ToolResult{
            Content: []TextContent{{Type: "text", Text: result}},
        }, nil

    case "priority_inbox":
        result, err := intelligentServer.HandlePriorityInbox(params.Arguments)
        if err != nil {
            return nil, err
        }
        return ToolResult{
            Content: []TextContent{{Type: "text", Text: result}},
        }, nil

    case "smart_filter":
        result, err := intelligentServer.HandleSmartFilter(params.Arguments)
        if err != nil {
            return nil, err
        }
        return ToolResult{
            Content: []TextContent{{Type: "text", Text: result}},
        }, nil

    case "analyze_priority":
        result, err := intelligentServer.HandleAnalyzePriority(params.Arguments)
        if err != nil {
            return nil, err
        }
        return ToolResult{
            Content: []TextContent{{Type: "text", Text: result}},
        }, nil
    }
}
```

### Sync IMAP Emails to Database

```go
import (
    "email-mcp-server/ai"
    "email-mcp-server/storage"
    "email-mcp-server/utils"
)

func syncIMAPToDatabase(imapEmails []EmailMessage, accountID string) error {
    db := intelligentServer.GetDatabase() // You'll need to add this getter
    classifier := intelligentServer.GetClassifier()
    priority := intelligentServer.GetPriorityEngine()

    for _, imapEmail := range imapEmails {
        // Convert to storage format
        email := &storage.Email{
            ID:          utils.GenerateEmailID(accountID, imapEmail.MessageID, imapEmail.Date),
            AccountID:   accountID,
            From:        imapEmail.From,
            To:          strings.Join(imapEmail.To, ", "),
            Subject:     imapEmail.Subject,
            BodySnippet: truncate(imapEmail.Body, 500),
            ReceivedAt:  imapEmail.Date,
            Read:        hasFlag(imapEmail.Flags, "\\Seen"),
            Starred:     hasFlag(imapEmail.Flags, "\\Flagged"),
        }

        // Save to database
        if err := db.CreateEmail(email); err != nil {
            continue // Skip if already exists
        }

        // Classify
        aiEmail := utils.EmailToAIEmail(email)
        classification, _ := classifier.Classify(aiEmail)
        classifier.SaveClassification(classification)

        // Calculate priority
        priorityScore, _ := priority.CalculatePriority(aiEmail)
        priority.SavePriority(priorityScore)
    }

    return nil
}
```

---

## API Reference

### Classification Categories

| Category | Description | Common Patterns |
|----------|-------------|-----------------|
| `work` | Work-related emails | Company domain, project keywords |
| `personal` | Personal emails | Non-company domains |
| `promotions` | Marketing emails | Sale, offer, discount keywords |
| `invoice` | Financial emails | Invoice, payment, receipt |
| `newsletters` | Subscriptions | Newsletter, digest, List-Unsubscribe header |
| `urgent` | Time-sensitive | Urgent, ASAP, critical keywords |

### Priority Score Ranges

| Score | Level | Badge | Action |
|-------|-------|-------|--------|
| 90-100 | Critical | 🔴 | Immediate attention required |
| 70-89 | High | 🟠 | Address today |
| 40-69 | Medium | 🟡 | Normal queue |
| 20-39 | Low | 🟢 | Can wait |
| 0-19 | Minimal | ⚪ | Low importance |

### Error Codes

| Code | Message | Solution |
|------|---------|----------|
| `DB_ERROR` | Database connection failed | Check DB_PATH environment variable |
| `CONFIG_ERROR` | Invalid configuration | Validate JSON syntax in config files |
| `CLASSIFY_ERROR` | Classification failed | Check email has required fields (from, subject) |
| `PRIORITY_ERROR` | Priority calculation failed | Ensure database is initialized |

---

## Best Practices

### 1. Regular Sync

Sync IMAP emails to database regularly for best results:

```go
// Run every 15 minutes
ticker := time.NewTicker(15 * time.Minute)
go func() {
    for range ticker.C {
        syncIMAPToDatabase(getNewEmails(), accountID)
    }
}()
```

### 2. Cache Management

Classification results are cached for 24h. Clear cache if rules change:

```go
classifier.ClearCache()
```

### 3. VIP Management

Update VIP senders based on user actions:

```go
// User stars email from unknown sender 3+ times
priorityEngine.UpdateVIPStatus("newsender@example.com", true)
```

### 4. Custom Categories

Add domain-specific categories in your config:

```json
"customers_vip": {
  "description": "VIP customer emails",
  "conditions": [...],
  "priority_boost": 50,
  "tags": ["vip", "customer", "urgent"]
}
```

---

## Troubleshooting

### Issue: Low accuracy in classification

**Solution**: Add more specific rules to `classification_rules` in config.

### Issue: Wrong priority scores

**Solution**: Check `vip_senders` and `urgent_keywords` lists. Adjust `category_priority` weights.

### Issue: Database locked errors

**Solution**: Ensure only one process accesses the database. Use connection pooling.

### Issue: High memory usage

**Solution**: Clear classification cache periodically. Reduce database cache size.

---

## Performance Tips

1. **Batch Processing**: Process emails in batches of 100
2. **Lazy Loading**: Only load email bodies when needed
3. **Index Optimization**: Database auto-creates indexes, ensure they're used
4. **Cache TTL**: Adjust cache TTL based on email volume
5. **Concurrent Processing**: Use goroutines for independent operations

---

## Support & Contribution

- **Issues**: https://github.com/scopweb/mcp-server-go-emails/issues
- **Docs**: See [ARCHITECTURE.md](ARCHITECTURE.md) for technical details
- **Roadmap**: See [ROADMAP.md](../ROADMAP.md) for planned features

---

**Last Updated**: 2025-01-13
**Version**: 0.5.0-MVP
