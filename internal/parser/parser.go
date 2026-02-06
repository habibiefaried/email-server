package parser

import (
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
)

// Email represents a parsed email with extracted fields and attachments
type Email struct {
	From        string
	To          string
	Subject     string
	Date        string
	Body        string // Plain text body
	HTMLBody    string // HTML body
	Attachments []Attachment
	RawContent  string
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Parse extracts headers, body, and attachments from raw email content
func Parse(rawContent string) (*Email, error) {
	lines := strings.Split(rawContent, "\n")
	var headerStart int
	for i, line := range lines {
		if strings.Contains(line, "Received:") || strings.Contains(line, "MIME-Version:") ||
			strings.Contains(line, "From:") && strings.Contains(line, "<") {
			headerStart = i
			break
		}
	}

	emailContent := strings.Join(lines[headerStart:], "\n")
	r := strings.NewReader(emailContent)
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return nil, err
	}

	parsed := &Email{
		From:       msg.Header.Get("From"),
		To:         msg.Header.Get("To"),
		Subject:    msg.Header.Get("Subject"),
		Date:       msg.Header.Get("Date"),
		RawContent: rawContent,
	}

	if parsed.From == "" || parsed.To == "" {
		for _, line := range lines[:min(20, len(lines))] {
			if strings.HasPrefix(line, "From:") && !strings.Contains(line, "<") {
				parsed.From = strings.TrimPrefix(line, "From:")
				parsed.From = strings.TrimSpace(parsed.From)
			}
			if strings.HasPrefix(line, "To:") {
				parsed.To = strings.TrimPrefix(line, "To:")
				parsed.To = strings.TrimSpace(parsed.To)
			}
		}
	}

	mediaType, params, _ := mime.ParseMediaType(msg.Header.Get("Content-Type"))

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		reader := multipart.NewReader(msg.Body, boundary)

		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}

			partMediaType, partParams, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
			disposition := part.Header.Get("Content-Disposition")
			filename := partParams["filename"]
			if filename == "" {
				filename = partParams["name"]
			}

			body, _ := io.ReadAll(part)

			transferEncoding := strings.ToLower(part.Header.Get("Content-Transfer-Encoding"))
			if (strings.HasPrefix(disposition, "attachment") ||
				(strings.HasPrefix(disposition, "inline") && filename != "")) &&
				filename != "" {
				var data []byte
				if strings.Contains(transferEncoding, "base64") {
					data, _ = base64.StdEncoding.DecodeString(string(body))
				} else {
					data = body
				}
				parsed.Attachments = append(parsed.Attachments, Attachment{
					Filename:    filename,
					ContentType: partMediaType,
					Data:        data,
				})
			} else if strings.HasPrefix(partMediaType, "text/") {
				var decodedBody []byte
				if strings.Contains(transferEncoding, "base64") {
					decodedBody, _ = base64.StdEncoding.DecodeString(string(body))
				} else {
					decodedBody = body
				}
				if strings.HasPrefix(partMediaType, "text/html") {
					parsed.HTMLBody = string(decodedBody)
				} else if strings.HasPrefix(partMediaType, "text/plain") {
					parsed.Body = string(decodedBody)
				}
			}
		}
	} else {
		body, _ := io.ReadAll(msg.Body)
		transferEncoding := strings.ToLower(msg.Header.Get("Content-Transfer-Encoding"))
		var decodedBody []byte
		if strings.Contains(transferEncoding, "base64") {
			decodedBody, _ = base64.StdEncoding.DecodeString(string(body))
		} else {
			decodedBody = body
		}
		if strings.HasPrefix(mediaType, "text/html") {
			parsed.HTMLBody = string(decodedBody)
		} else {
			parsed.Body = string(decodedBody)
		}
	}

	return parsed, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
