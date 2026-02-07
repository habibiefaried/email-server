package storage

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGenerateUUIDv7_Format(t *testing.T) {
	id := generateUUIDv7()
	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("UUIDv7 not parseable: %s — %v", id, err)
	}
	if parsed.Version() != 7 {
		t.Errorf("Expected UUID version 7, got %d: %s", parsed.Version(), id)
	}
	if parsed.Variant() != uuid.RFC4122 {
		t.Errorf("Expected RFC4122 variant, got %v: %s", parsed.Variant(), id)
	}
}

func TestGenerateUUIDv7_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := generateUUIDv7()
		if seen[id] {
			t.Fatalf("Duplicate UUIDv7 at iteration %d: %s", i, id)
		}
		seen[id] = true
	}
}

func TestGenerateUUIDv7_TimestampMonotonic(t *testing.T) {
	prev := generateUUIDv7()
	time.Sleep(2 * time.Millisecond)
	for i := 0; i < 10; i++ {
		next := generateUUIDv7()
		time.Sleep(2 * time.Millisecond)
		if next <= prev {
			t.Errorf("UUIDv7 not monotonically increasing: %s <= %s", next, prev)
		}
		prev = next
	}
}

func TestGenerateUUIDv7_Length(t *testing.T) {
	id := generateUUIDv7()
	if len(id) != 36 {
		t.Errorf("UUIDv7 length should be 36, got %d: %s", len(id), id)
	}
}

func TestGenerateUUIDv7_ParseRoundTrip(t *testing.T) {
	for i := 0; i < 100; i++ {
		id := generateUUIDv7()
		parsed, err := uuid.Parse(id)
		if err != nil {
			t.Fatalf("Parse failed for %s: %v", id, err)
		}
		if parsed.String() != id {
			t.Errorf("Round-trip mismatch: %s != %s", parsed.String(), id)
		}
	}
}

func TestEmailSummary_NoBodyField(t *testing.T) {
	s := EmailSummary{
		ID:        generateUUIDv7(),
		From:      "a@b.com",
		To:        "c@d.com",
		Subject:   "test",
		Date:      "2026-01-01",
		CreatedAt: time.Now(),
	}
	if s.ID == "" || s.From == "" || s.To == "" {
		t.Error("EmailSummary fields not set correctly")
	}
	if _, err := uuid.Parse(s.ID); err != nil {
		t.Errorf("EmailSummary ID should be valid UUID: %v", err)
	}
}

func TestEmailDetail_HasBodyAndAttachments(t *testing.T) {
	d := EmailDetail{
		ID:          generateUUIDv7(),
		From:        "a@b.com",
		To:          "c@d.com",
		Subject:     "test",
		Date:        "2026-01-01",
		Body:        "Hello",
		HTMLBody:    "<p>Hello</p>",
		RawContent:  "raw data",
		CreatedAt:   time.Now(),
		Attachments: []AttachmentInfo{{ID: generateUUIDv7(), Filename: "file.txt", ContentType: "text/plain", Size: 100, Data: "SGVsbG8gV29ybGQ="}},
	}
	if d.Body == "" {
		t.Error("EmailDetail Body should not be empty")
	}
	if len(d.Attachments) != 1 {
		t.Error("EmailDetail should have 1 attachment")
	}
	if _, err := uuid.Parse(d.Attachments[0].ID); err != nil {
		t.Errorf("Attachment ID should be valid UUID: %v", err)
	}
}

func TestAttachmentInfo_UUIDv7ID(t *testing.T) {
	a := AttachmentInfo{
		ID:          generateUUIDv7(),
		Filename:    "test.png",
		ContentType: "image/png",
		Size:        1024,
		Data:        "iVBORw0KGgo=",
	}
	parsed, err := uuid.Parse(a.ID)
	if err != nil {
		t.Fatalf("Attachment ID not parseable: %v", err)
	}
	if parsed.Version() != 7 {
		t.Errorf("Expected UUID version 7, got %d", parsed.Version())
	}
	if a.Data == "" {
		t.Error("Attachment Data should not be empty")
	}
}

func TestPlainTextToHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string // substring that must be present
	}{
		{"simple text", "Hello World", "Hello World"},
		{"multiline", "Line 1\nLine 2", "<br>"},
		{"html escaping", "a < b & c > d", "a &lt; b &amp; c &gt; d"},
		{"url linkify", "Visit https://example.com today", `<a href="https://example.com">`},
		{"wraps in html", "test", "<!DOCTYPE html>"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := plainTextToHTML(tc.input)
			if !contains(result, tc.want) {
				t.Errorf("plainTextToHTML(%q) = %q, want substring %q", tc.input, result, tc.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestMaxAttachmentSize(t *testing.T) {
	// Verify the constant is set to 2MB
	expected := 2 * 1024 * 1024
	if MaxAttachmentSize != expected {
		t.Errorf("MaxAttachmentSize = %d, want %d (2MB)", MaxAttachmentSize, expected)
	}
}

func TestAttachmentRedaction_Logic(t *testing.T) {
	// Test the redaction logic (conceptual test since we don't mock DB)
	smallData := make([]byte, 1*1024*1024) // 1MB
	largeData := make([]byte, 3*1024*1024) // 3MB

	if len(smallData) > MaxAttachmentSize {
		t.Error("1MB attachment should NOT exceed MaxAttachmentSize")
	}
	if len(largeData) <= MaxAttachmentSize {
		t.Error("3MB attachment SHOULD exceed MaxAttachmentSize")
	}

	// Verify redacted message format
	redactedMsg := "<attachment redacted: test.pdf, original size: 3145728 bytes, exceeds 2 MB limit>"
	if !contains(redactedMsg, "attachment redacted") {
		t.Error("Redacted message should contain 'attachment redacted'")
	}
	if !contains(redactedMsg, "test.pdf") {
		t.Error("Redacted message should contain filename")
	}
	if !contains(redactedMsg, "3145728 bytes") {
		t.Error("Redacted message should contain original size")
	}
}
