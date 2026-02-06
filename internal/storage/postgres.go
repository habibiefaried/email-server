package storage

import (
	"database/sql"
	"log"
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

// AttachmentInfo represents attachment metadata
type AttachmentInfo struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
}

// GetInbox fetches email summaries for a recipient (10 per page, offset only)
func (ps *PostgresStorage) GetInbox(address string, offset int) ([]EmailSummary, error) {
	if offset < 0 {
		offset = 0
	}
	rows, err := ps.db.Query(`
		SELECT id, "from", "to", COALESCE(subject, ''), COALESCE(date, ''), created_at
		FROM email
		WHERE "to" = $1
		ORDER BY created_at DESC
		LIMIT 10 OFFSET $2
	`, address, offset)
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

	attRows, err := ps.db.Query(`
		SELECT id, filename, COALESCE(content_type, ''), COALESCE(length(data), 0)
		FROM attachment WHERE email_id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	defer attRows.Close()

	email.Attachments = make([]AttachmentInfo, 0)
	for attRows.Next() {
		var att AttachmentInfo
		if err := attRows.Scan(&att.ID, &att.Filename, &att.ContentType, &att.Size); err != nil {
			return nil, err
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
