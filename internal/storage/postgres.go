package storage

import (
	"database/sql"
	"fmt"
	"log"
	"time"

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
	emailTableSQL := `
	CREATE TABLE IF NOT EXISTS email (
		id SERIAL PRIMARY KEY,
		"from" TEXT NOT NULL,
		"to" TEXT NOT NULL,
		subject TEXT,
		date TEXT,
		body TEXT,
		raw_content TEXT,
		headers TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	attachmentTableSQL := `
	CREATE TABLE IF NOT EXISTS attachment (
		id SERIAL PRIMARY KEY,
		email_id INTEGER NOT NULL REFERENCES email(id) ON DELETE CASCADE,
		filename TEXT NOT NULL,
		content_type TEXT,
		data BYTEA,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	// Create email table
	if _, err := ps.db.Exec(emailTableSQL); err != nil {
		return err
	}

	// Create attachment table
	if _, err := ps.db.Exec(attachmentTableSQL); err != nil {
		return err
	}

	// Create indexes
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

// Save saves an email and its attachments to postgres
func (ps *PostgresStorage) Save(email Email) (string, error) {
	// Parse the email content
	parsed, err := parser.Parse(email.Content)
	if err != nil {
		// If parsing fails, still save with basic info
		parsed = &parser.Email{
			From:       email.From,
			To:         email.To,
			RawContent: email.Content,
		}
	}

	// Start a transaction
	tx, err := ps.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Insert email record
	var emailID int
	err = tx.QueryRow(
		`INSERT INTO email ("from", "to", subject, date, body, raw_content)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		parsed.From,
		parsed.To,
		parsed.Subject,
		parsed.Date,
		parsed.Body,
		parsed.RawContent,
	).Scan(&emailID)

	if err != nil {
		return "", err
	}

	// Insert attachments
	for _, att := range parsed.Attachments {
		_, err := tx.Exec(
			`INSERT INTO attachment (email_id, filename, content_type, data)
			 VALUES ($1, $2, $3, $4)`,
			emailID,
			att.Filename,
			att.ContentType,
			att.Data,
		)
		if err != nil {
			return "", err
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("postgres_id_%d_%d", emailID, time.Now().UnixNano())
	log.Printf("Email saved to postgres: id=%d, from=%s, to=%s, attachments=%d",
		emailID, parsed.From, parsed.To, len(parsed.Attachments))

	return filename, nil
}

// Close closes the database connection
func (ps *PostgresStorage) Close() error {
	if ps.db != nil {
		return ps.db.Close()
	}
	return nil
}
