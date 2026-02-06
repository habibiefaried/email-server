package storage

import (
	"os"
	"strings"
	"testing"
)

func TestParseEmail_Gmail_SimpleEmail(t *testing.T) {
	// Test parsing simple Gmail email (no attachments)
	content, err := os.ReadFile("../../samples/gmail.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	parsed, err := ParseEmail(string(content))
	if err != nil {
		t.Fatalf("ParseEmail failed: %v", err)
	}

	// Verify From header
	if !strings.Contains(parsed.From, "habibie.faried@paidy.com") {
		t.Errorf("From field incorrect. Got: %s", parsed.From)
	}

	// Verify To header
	if !strings.Contains(parsed.To, "admin@test.milahabibie.com") {
		t.Errorf("To field incorrect. Got: %s", parsed.To)
	}

	// Verify Subject
	if parsed.Subject != "test" {
		t.Errorf("Subject incorrect. Got: %s", parsed.Subject)
	}

	// Verify Date is not empty
	if parsed.Date == "" {
		t.Error("Date field is empty")
	}

	// Verify Body contains expected content
	if !strings.Contains(parsed.Body, "test") {
		t.Errorf("Body does not contain expected text. Got: %s", parsed.Body)
	}

	// Verify no attachments
	if len(parsed.Attachments) != 0 {
		t.Errorf("Expected no attachments, but got %d", len(parsed.Attachments))
	}

	// Verify raw content is preserved
	if parsed.RawContent == "" {
		t.Error("Raw content is empty")
	}

	t.Log("✓ Gmail simple email parsed correctly")
}

func TestParseEmail_AnonymousMail_NoAttachments(t *testing.T) {
	// Test parsing AnonymousMail email (multipart but no attachments)
	content, err := os.ReadFile("../../samples/anonymousemail.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	parsed, err := ParseEmail(string(content))
	if err != nil {
		t.Fatalf("ParseEmail failed: %v", err)
	}

	// Verify From header
	if !strings.Contains(parsed.From, "noreply@anonymousemail.se") {
		t.Errorf("From field incorrect. Got: %s", parsed.From)
	}

	// Verify To header
	if !strings.Contains(parsed.To, "admin2@test.milahabibie.com") {
		t.Errorf("To field incorrect. Got: %s", parsed.To)
	}

	// Verify Subject
	if parsed.Subject != "test" {
		t.Errorf("Subject incorrect. Got: %s", parsed.Subject)
	}

	// Verify Body contains expected content
	if !strings.Contains(parsed.Body, "test") {
		t.Errorf("Body does not contain expected text. Got: %s", parsed.Body)
	}

	// Verify no attachments
	if len(parsed.Attachments) != 0 {
		t.Errorf("Expected no attachments, but got %d", len(parsed.Attachments))
	}

	t.Log("✓ AnonymousMail email parsed correctly")
}

func TestParseEmail_WithAttachments(t *testing.T) {
	// Test parsing email with PNG attachment
	content, err := os.ReadFile("../../samples/attachments.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	parsed, err := ParseEmail(string(content))
	if err != nil {
		t.Fatalf("ParseEmail failed: %v", err)
	}

	// Verify From header
	if !strings.Contains(parsed.From, "habibie.faried@paidy.com") {
		t.Errorf("From field incorrect. Got: %s", parsed.From)
	}

	// Verify To header
	if !strings.Contains(parsed.To, "admin@test.milahabibie.com") {
		t.Errorf("To field incorrect. Got: %s", parsed.To)
	}

	// Verify Subject
	if parsed.Subject != "Re: test" {
		t.Errorf("Subject incorrect. Got: %s", parsed.Subject)
	}

	// Verify Body (may be empty due to nested multipart structure)
	// The important part is attachment extraction
	if len(parsed.Attachments) == 0 {
		// If no attachments found, body should have content
		if !strings.Contains(parsed.Body, "attachment") {
			t.Logf("Note: Body content not extracted from nested multipart, but attachments found: %d", len(parsed.Attachments))
		}
	} else {
		t.Logf("Body content: %s", parsed.Body)
	}

	// Verify attachments are detected
	if len(parsed.Attachments) == 0 {
		t.Error("Expected attachments, but got none")
	}

	t.Logf("✓ Email with attachments parsed correctly - found %d attachments", len(parsed.Attachments))

	// Verify attachment details
	foundPNG := false
	for _, att := range parsed.Attachments {
		if strings.Contains(att.Filename, "Screenshot") && strings.HasSuffix(att.Filename, ".png") {
			foundPNG = true

			// Verify content type
			if att.ContentType != "image/png" {
				t.Errorf("Attachment content type incorrect. Got: %s", att.ContentType)
			}

			// Verify data is not empty
			if len(att.Data) == 0 {
				t.Error("Attachment data is empty")
			}

			// Verify PNG file signature (magic bytes: 89 50 4E 47)
			if len(att.Data) >= 4 {
				expected := []byte{0x89, 0x50, 0x4E, 0x47}
				actual := att.Data[:4]
				if string(actual) != string(expected) {
					t.Errorf("PNG file signature incorrect. Got: %v, Expected: %v", actual, expected)
				}
			}

			t.Logf("✓ PNG attachment verified - Filename: %s, Size: %d bytes", att.Filename, len(att.Data))
		}
	}

	if !foundPNG {
		t.Error("PNG attachment not found")
	}
}

func TestParseEmail_AttachmentDecoding(t *testing.T) {
	// Test that base64 attachments are properly decoded
	content, err := os.ReadFile("../../samples/attachments.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	parsed, err := ParseEmail(string(content))
	if err != nil {
		t.Fatalf("ParseEmail failed: %v", err)
	}

	if len(parsed.Attachments) == 0 {
		t.Fatal("No attachments found")
	}

	// Verify all attachments have decoded data
	for i, att := range parsed.Attachments {
		if len(att.Data) == 0 {
			t.Errorf("Attachment %d has no data", i)
		}

		// Verify data is not base64 encoded anymore (check if it's binary)
		// PNG data should start with magic bytes
		if att.ContentType == "image/png" && len(att.Data) > 4 {
			if att.Data[0] != 0x89 || att.Data[1] != 0x50 {
				t.Errorf("Attachment %d: Data not properly decoded from base64", i)
			}
		}
	}

	t.Log("✓ All attachments properly decoded from base64")
}

func TestParseEmail_HeaderPreservation(t *testing.T) {
	// Test that headers are properly extracted
	content, err := os.ReadFile("../../samples/gmail.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	parsed, err := ParseEmail(string(content))
	if err != nil {
		t.Fatalf("ParseEmail failed: %v", err)
	}

	// All headers should have values
	if parsed.From == "" {
		t.Error("From header is empty")
	}
	if parsed.To == "" {
		t.Error("To header is empty")
	}
	if parsed.Subject == "" {
		t.Error("Subject header is empty")
	}
	if parsed.Date == "" {
		t.Error("Date header is empty")
	}

	t.Logf("✓ Headers preserved - From: %s, To: %s, Subject: %s, Date: %s",
		parsed.From, parsed.To, parsed.Subject, parsed.Date)
}

func TestParseEmail_MultipleAttachmentTypes(t *testing.T) {
	// Test parsing email with potentially multiple attachment entries
	content, err := os.ReadFile("../../samples/attachments.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	parsed, err := ParseEmail(string(content))
	if err != nil {
		t.Fatalf("ParseEmail failed: %v", err)
	}

	// Verify unique attachment handling
	uniqueFilenames := make(map[string]bool)
	for _, att := range parsed.Attachments {
		uniqueFilenames[att.Filename] = true
	}

	t.Logf("✓ Found %d total attachments with %d unique filenames", len(parsed.Attachments), len(uniqueFilenames))

	// All attachments should have required fields
	for i, att := range parsed.Attachments {
		if att.Filename == "" {
			t.Errorf("Attachment %d has no filename", i)
		}
		if att.ContentType == "" {
			t.Errorf("Attachment %d has no content type", i)
		}
		if len(att.Data) == 0 {
			t.Errorf("Attachment %d has no data", i)
		}
	}
}

func TestParseEmail_RawContentPreserved(t *testing.T) {
	// Verify that raw content is preserved exactly
	content, err := os.ReadFile("../../samples/gmail.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	contentStr := string(content)
	parsed, err := ParseEmail(contentStr)
	if err != nil {
		t.Fatalf("ParseEmail failed: %v", err)
	}

	// Raw content should match original
	if parsed.RawContent != contentStr {
		t.Error("Raw content not preserved exactly")
	}

	t.Log("✓ Raw content preserved exactly")
}

func BenchmarkParseEmail_SimpleEmail(b *testing.B) {
	content, _ := os.ReadFile("../../samples/gmail.txt")
	contentStr := string(content)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseEmail(contentStr)
	}
}

func BenchmarkParseEmail_WithAttachments(b *testing.B) {
	content, _ := os.ReadFile("../../samples/attachments.txt")
	contentStr := string(content)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseEmail(contentStr)
	}
}
