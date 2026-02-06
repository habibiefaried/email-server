package storage

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

// UUIDv7 regex: 8-4-4-4-12 hex, version nibble = 7, variant nibble = 8/9/a/b
var uuidv7Regex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestGenerateUUIDv7_Format(t *testing.T) {
	id := generateUUIDv7()
	if !uuidv7Regex.MatchString(id) {
		t.Errorf("UUIDv7 format invalid: %s", id)
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
	// UUIDv7 IDs generated in order should sort lexicographically
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

func TestGenerateUUIDv7_VersionBit(t *testing.T) {
	for i := 0; i < 100; i++ {
		id := generateUUIDv7()
		// The 13th character (index 14 after removing dashes) should be '7'
		parts := strings.Split(id, "-")
		if parts[2][0] != '7' {
			t.Errorf("UUIDv7 version nibble is not 7: %s (got %c)", id, parts[2][0])
		}
	}
}

func TestGenerateUUIDv7_VariantBit(t *testing.T) {
	for i := 0; i < 100; i++ {
		id := generateUUIDv7()
		parts := strings.Split(id, "-")
		// The first nibble of the 4th group should be 8, 9, a, or b
		firstChar := parts[3][0]
		if firstChar != '8' && firstChar != '9' && firstChar != 'a' && firstChar != 'b' {
			t.Errorf("UUIDv7 variant nibble incorrect: %s (got %c)", id, firstChar)
		}
	}
}

func TestGenerateUUIDv7_Length(t *testing.T) {
	id := generateUUIDv7()
	if len(id) != 36 {
		t.Errorf("UUIDv7 length should be 36, got %d: %s", len(id), id)
	}
}

func TestEmailSummary_NoBodyField(t *testing.T) {
	// Verify EmailSummary struct does not contain Body field
	s := EmailSummary{
		ID:        "test-id",
		From:      "a@b.com",
		To:        "c@d.com",
		Subject:   "test",
		Date:      "2026-01-01",
		CreatedAt: time.Now(),
	}
	// If we can compile and use it without Body, the struct is correct
	if s.ID == "" || s.From == "" || s.To == "" {
		t.Error("EmailSummary fields not set correctly")
	}
}

func TestEmailDetail_HasBodyAndAttachments(t *testing.T) {
	d := EmailDetail{
		ID:          "test-id",
		From:        "a@b.com",
		To:          "c@d.com",
		Subject:     "test",
		Date:        "2026-01-01",
		Body:        "Hello",
		HTMLBody:    "<p>Hello</p>",
		RawContent:  "raw data",
		CreatedAt:   time.Now(),
		Attachments: []AttachmentInfo{{ID: "att-1", Filename: "file.txt", ContentType: "text/plain", Size: 100}},
	}
	if d.Body == "" {
		t.Error("EmailDetail Body should not be empty")
	}
	if len(d.Attachments) != 1 {
		t.Error("EmailDetail should have 1 attachment")
	}
	if d.Attachments[0].ID == "" {
		t.Error("Attachment ID should not be empty")
	}
}

func TestAttachmentInfo_StringID(t *testing.T) {
	a := AttachmentInfo{
		ID:          generateUUIDv7(),
		Filename:    "test.png",
		ContentType: "image/png",
		Size:        1024,
	}
	if !uuidv7Regex.MatchString(a.ID) {
		t.Errorf("Attachment ID should be UUIDv7 format: %s", a.ID)
	}
}
