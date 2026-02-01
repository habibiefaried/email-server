package storage

import (
	"os"
	"strings"
	"testing"
)

func TestFileStorage_Save(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStorage(dir)
	email := Email{
		From:    "alice@example.com",
		To:      "bob@example.com",
		Content: "Hello, Bob!",
	}
	filename, err := fs.Save(email)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "alice@example.com") || !strings.Contains(content, "bob@example.com") || !strings.Contains(content, "Hello, Bob!") {
		t.Errorf("Saved file content incorrect: %s", content)
	}
}
