package storage

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/habibiefaried/email-server/internal/parser"
	_ "github.com/lib/pq"
)

// PostgresStorage implements Storage interface with Postgres backend
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage creates a new postgres storage instance
// dsn format: "user=username password=pass dbname=emaildb host=localhost port=5432 sslmode=disable"
func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	ps := &PostgresStorage{db: db}
	if err := ps.createTables(); err != nil {
		return nil, err
	}

	log.Printf("Connected to Postgres database")
	return ps, nil
}

// createTables creates the email and attachment tables if they don't exist
func (ps *PostgresStorage) createTables() error {
	// Migration: detect old schema (SERIAL integer id) and drop tables
	var colType string
	err := ps.db.QueryRow(`
		SELECT data_type FROM information_schema.columns
		WHERE table_name = 'email' AND column_name = 'id'
	`).Scan(&colType)
	if err == nil && (colType == "integer" || colType == "text") {
		log.Printf("Migrating database schema to native UUID columns...")
		ps.db.Exec(`DROP TABLE IF EXISTS attachment CASCADE`)
		ps.db.Exec(`DROP TABLE IF EXISTS email CASCADE`)
	}

	emailTableSQL := `
	CREATE TABLE IF NOT EXISTS email (
		id UUID PRIMARY KEY,
		"from" TEXT NOT NULL,
		"to" TEXT NOT NULL,
		subject TEXT,
		date TEXT,
		body TEXT,
		html_body TEXT,
		raw_content TEXT,
		headers TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	attachmentTableSQL := `
	CREATE TABLE IF NOT EXISTS attachment (
		id UUID PRIMARY KEY,
		email_id UUID NOT NULL REFERENCES email(id) ON DELETE CASCADE,
		filename TEXT NOT NULL,
		content_type TEXT,
		data BYTEA,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := ps.db.Exec(emailTableSQL); err != nil {
		return err
	}
	if _, err := ps.db.Exec(attachmentTableSQL); err != nil {
		return err
	}

	indexSQL := `
	CREATE INDEX IF NOT EXISTS idx_email_from ON email("from");
	CREATE INDEX IF NOT EXISTS idx_email_to ON email("to");
	CREATE INDEX IF NOT EXISTS idx_email_created_at ON email(created_at);
	CREATE INDEX IF NOT EXISTS idx_attachment_email_id ON attachment(email_id);`

	if _, err := ps.db.Exec(indexSQL); err != nil {
		return err
	}

	return nil
}

// generateUUIDv7 generates a UUIDv7 using github.com/google/uuid
func generateUUIDv7() string {
	id, err := uuid.NewV7()
	if err != nil {
		// Fallback to V4 if V7 fails (should never happen)
		return uuid.New().String()
	}
	return id.String()
}

// Save saves an email and its attachments to postgres
// Base64 content is decoded by the parser BEFORE inserting into the database
func (ps *PostgresStorage) Save(email Email) (string, error) {
	// Parse the email content (base64 decoding happens here)
	parsed, err := parser.Parse(email.Content)
	if err != nil {
		parsed = &parser.Email{
			From:       email.From,
			To:         email.To,
			RawContent: email.Content,
		}
	}

	// Ensure body and html_body are not empty by generating HTML from raw content if needed
	if parsed.Body == "" && parsed.HTMLBody == "" && parsed.RawContent != "" {
		if reparsed, err := parser.Parse(parsed.RawContent); err == nil {
			if reparsed.HTMLBody != "" {
				parsed.HTMLBody = reparsed.HTMLBody
				parsed.Body = reparsed.HTMLBody
			} else if reparsed.Body != "" {
				parsed.Body = plainTextToHTML(reparsed.Body)
				parsed.HTMLBody = parsed.Body
			}
		}
	}

	// If only Body is set, generate HTML version
	if parsed.Body != "" && parsed.HTMLBody == "" {
		parsed.HTMLBody = plainTextToHTML(parsed.Body)
	}

	// If only HTMLBody is set, use it as Body too
	if parsed.HTMLBody != "" && parsed.Body == "" {
		parsed.Body = parsed.HTMLBody
	}

	tx, err := ps.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	emailID := generateUUIDv7()
	_, err = tx.Exec(
		`INSERT INTO email (id, "from", "to", subject, date, body, html_body, raw_content)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		emailID,
		parsed.From,
		parsed.To,
		parsed.Subject,
		parsed.Date,
		parsed.Body,
		parsed.HTMLBody,
		parsed.RawContent,
	)
	if err != nil {
		return "", err
	}

	for _, att := range parsed.Attachments {
		attID := generateUUIDv7()
		_, err := tx.Exec(
			`INSERT INTO attachment (id, email_id, filename, content_type, data)
			 VALUES ($1, $2, $3, $4, $5)`,
			attID,
			emailID,
			att.Filename,
			att.ContentType,
			att.Data,
		)
		if err != nil {
			return "", err
		}
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	log.Printf("Email saved to postgres: id=%s, from=%s, to=%s, attachments=%d",
		emailID, parsed.From, parsed.To, len(parsed.Attachments))

	return emailID, nil
}

// EmailSummary represents email metadata for inbox listing (no body/attachments)
type EmailSummary struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Date      string    `json:"date"`
	CreatedAt time.Time `json:"created_at"`
}

// EmailDetail represents a full email with body and attachment metadata
type EmailDetail struct {
	ID          string           `json:"id"`
	From        string           `json:"from"`
	To          string           `json:"to"`
	Subject     string           `json:"subject"`
	Date        string           `json:"date"`
	Body        string           `json:"body"`
	HTMLBody    string           `json:"html_body,omitempty"`
	RawContent  string           `json:"raw_content,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	Attachments []AttachmentInfo `json:"attachments"`
}

// AttachmentInfo represents attachment metadata and data
type AttachmentInfo struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
	Data        string `json:"data"` // base64-encoded attachment data
}

// GetInbox fetches email summaries for a recipient (5 per page).
// The page parameter is 1-based: page=1 returns rows 1-5, page=2 returns rows 6-10, etc.
func (ps *PostgresStorage) GetInbox(address string, page int) ([]EmailSummary, error) {
	if page < 1 {
		page = 1
	}
	const pageSize = 5
	sqlOffset := (page - 1) * pageSize
	rows, err := ps.db.Query(`
		SELECT id, "from", "to", COALESCE(subject, ''), COALESCE(date, ''), created_at
		FROM email
		WHERE "to" = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, address, pageSize, sqlOffset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emails := make([]EmailSummary, 0)
	for rows.Next() {
		var email EmailSummary
		if err := rows.Scan(&email.ID, &email.From, &email.To, &email.Subject, &email.Date, &email.CreatedAt); err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return emails, nil
}

// GetEmailByID fetches full email detail including body and attachments by UUIDv7
func (ps *PostgresStorage) GetEmailByID(id string) (*EmailDetail, error) {
	var email EmailDetail
	var htmlBody, rawContent sql.NullString
	err := ps.db.QueryRow(`
		SELECT id, "from", "to", COALESCE(subject, ''), COALESCE(date, ''),
		       COALESCE(body, ''), html_body, raw_content, created_at
		FROM email WHERE id = $1
	`, id).Scan(
		&email.ID, &email.From, &email.To, &email.Subject, &email.Date,
		&email.Body, &htmlBody, &rawContent, &email.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if htmlBody.Valid {
		email.HTMLBody = htmlBody.String
	}
	if rawContent.Valid {
		email.RawContent = rawContent.String
	}

	// If body is empty, try to parse raw_content and generate HTML
	if email.Body == "" && email.HTMLBody == "" && email.RawContent != "" {
		if parsed, err := parser.Parse(email.RawContent); err == nil {
			if parsed.HTMLBody != "" {
				email.HTMLBody = parsed.HTMLBody
				email.Body = parsed.HTMLBody
			} else if parsed.Body != "" {
				email.Body = plainTextToHTML(parsed.Body)
				email.HTMLBody = email.Body
			}
		}
	}

	// If we have HTMLBody but no Body, use HTMLBody as Body
	if email.Body == "" && email.HTMLBody != "" {
		email.Body = email.HTMLBody
	}

	// If we still only have plain text Body and no HTMLBody, convert to HTML
	if email.Body != "" && email.HTMLBody == "" {
		email.HTMLBody = plainTextToHTML(email.Body)
		email.Body = email.HTMLBody
	}

	attRows, err := ps.db.Query(`
		SELECT id, filename, COALESCE(content_type, ''), COALESCE(length(data), 0), data
		FROM attachment WHERE email_id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	defer attRows.Close()

	email.Attachments = make([]AttachmentInfo, 0)
	for attRows.Next() {
		var att AttachmentInfo
		var rawData []byte
		if err := attRows.Scan(&att.ID, &att.Filename, &att.ContentType, &att.Size, &rawData); err != nil {
			return nil, err
		}
		if rawData != nil {
			att.Data = base64.StdEncoding.EncodeToString(rawData)
		}
		email.Attachments = append(email.Attachments, att)
	}

	return &email, nil
}

// Close closes the database connection
func (ps *PostgresStorage) Close() error {
	if ps.db != nil {
		return ps.db.Close()
	}
	return nil
}

// plainTextToHTML converts plain text email body to HTML
func plainTextToHTML(text string) string {
	escaped := html.EscapeString(text)
	// Convert URLs to clickable links
	lines := strings.Split(escaped, "\n")
	for i, line := range lines {
		words := strings.Fields(line)
		for j, word := range words {
			if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
				words[j] = fmt.Sprintf(`<a href="%s">%s</a>`, word, word)
			}
		}
		lines[i] = strings.Join(words, " ")
	}
	body := strings.Join(lines, "<br>\n")
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="UTF-8"></head><body style="font-family:sans-serif;padding:16px;white-space:pre-wrap;">%s</body></html>`, body)
}
