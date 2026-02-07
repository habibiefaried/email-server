package main

import (
	"database/sql"
	"fmt"
	"html"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/habibiefaried/email-server/internal/parser"
	_ "github.com/lib/pq"
)

const (
	// MaxAttachmentSize is the maximum size (in bytes) for storing attachment data.
	// Attachments larger than this will be replaced with a redacted placeholder.
	MaxAttachmentSize = 2 * 1024 * 1024 // 2 MB
	// MaxEmailSize is the maximum size (in bytes) for displaying email raw content.
	// Emails larger than this will show a limit message instead of parsed content.
	MaxEmailSize = 512 * 1024 // 512 KB
)

type emailRow struct {
	ID         string
	RawContent string
}

func main() {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database")

	// Find emails with raw_content but empty/null body
	rows, err := db.Query(`
		SELECT id, raw_content
		FROM email
		WHERE raw_content IS NOT NULL
		  AND raw_content != ''
		  AND (body IS NULL OR body = '')
		  AND (html_body IS NULL OR html_body = '')
	`)
	if err != nil {
		log.Fatalf("Failed to query emails: %v", err)
	}
	defer rows.Close()

	var emails []emailRow
	for rows.Next() {
		var e emailRow
		if err := rows.Scan(&e.ID, &e.RawContent); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		emails = append(emails, e)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("Row iteration error: %v", err)
	}

	total := len(emails)
	if total == 0 {
		log.Println("No emails found with empty body. Nothing to do.")
		return
	}

	log.Printf("Found %d email(s) with raw_content but empty body. Processing...", total)

	updated := 0
	skipped := 0
	failed := 0

	for i, e := range emails {
		log.Printf("[%d/%d] Processing email %s...", i+1, total, e.ID)

		var body, htmlBody string

		// Check if email exceeds 512KB limit
		if len(e.RawContent) > MaxEmailSize {
			limitMsg := "Limit of this service is 512kb only"
			log.Printf("  Email exceeds 512KB limit (%d bytes), setting limit message", len(e.RawContent))
			body = limitMsg
			htmlBody = plainTextToHTML(limitMsg)
		} else {
			parsed, err := parser.Parse(e.RawContent)
			if err != nil {
				log.Printf("  SKIP: Failed to parse raw_content: %v", err)
				skipped++
				continue
			}

			if parsed.HTMLBody != "" {
				htmlBody = parsed.HTMLBody
				body = parsed.HTMLBody
			} else if parsed.Body != "" {
				htmlBody = plainTextToHTML(parsed.Body)
				body = htmlBody
			} else {
				log.Printf("  SKIP: Parser found no body or HTML in raw_content")
				skipped++
				continue
			}
		}

		// Update the email body and html_body
		result, err := db.Exec(`
			UPDATE email SET body = $1, html_body = $2 WHERE id = $3
		`, body, htmlBody, e.ID)
		if err != nil {
			log.Printf("  FAIL: Failed to update email: %v", err)
			failed++
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			log.Printf("  WARN: No rows affected for email %s", e.ID)
			failed++
			continue
		}

		log.Printf("  OK: Updated body (%d chars), html_body (%d chars)", len(body), len(htmlBody))

		// Also re-extract attachments if parser found any that aren't in DB yet (only for emails under size limit)
		if len(e.RawContent) <= MaxEmailSize {
			parsed, err := parser.Parse(e.RawContent)
			if err == nil && len(parsed.Attachments) > 0 {
				var attCount int
				db.QueryRow(`SELECT COUNT(*) FROM attachment WHERE email_id = $1`, e.ID).Scan(&attCount)

				if attCount == 0 {
					for _, att := range parsed.Attachments {
						attID := generateUUIDv7()
						attData := att.Data

						// Redact large attachments (>2MB) to avoid database bloat
						if len(attData) > MaxAttachmentSize {
							redactedMsg := fmt.Sprintf("<attachment redacted: %s, original size: %d bytes, exceeds %d MB limit>",
								att.Filename, len(attData), MaxAttachmentSize/(1024*1024))
							attData = []byte(redactedMsg)
							log.Printf("  Attachment redacted: %s (%d bytes > %d bytes limit)", att.Filename, len(att.Data), MaxAttachmentSize)
						}

						_, err := db.Exec(`
							INSERT INTO attachment (id, email_id, filename, content_type, data)
							VALUES ($1, $2, $3, $4, $5)
						`, attID, e.ID, att.Filename, att.ContentType, attData)
						if err != nil {
							log.Printf("  WARN: Failed to insert attachment %s: %v", att.Filename, err)
						} else {
							log.Printf("  Inserted attachment: %s (%s, %d bytes)", att.Filename, att.ContentType, len(att.Data))
						}
					}
				}
			}
		}

		updated++
	}

	log.Println("========================================")
	log.Printf("Done. Total: %d | Updated: %d | Skipped: %d | Failed: %d", total, updated, skipped, failed)

	// Also show remaining emails that still have empty body
	var remaining int
	db.QueryRow(`
		SELECT COUNT(*) FROM email
		WHERE raw_content IS NOT NULL AND raw_content != ''
		  AND (body IS NULL OR body = '')
		  AND (html_body IS NULL OR html_body = '')
	`).Scan(&remaining)
	if remaining > 0 {
		log.Printf("WARNING: %d email(s) still have empty body after processing", remaining)
	} else {
		log.Println("All emails with raw_content now have body populated.")
	}
}

func generateUUIDv7() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New().String()
	}
	return id.String()
}

// plainTextToHTML converts plain text email body to HTML
func plainTextToHTML(text string) string {
	escaped := html.EscapeString(text)
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
