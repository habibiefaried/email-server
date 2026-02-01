package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type FileStorage struct {
	Dir string
}

func NewFileStorage(dir string) *FileStorage {
	os.MkdirAll(dir, 0755)
	return &FileStorage{Dir: dir}
}

func (fs *FileStorage) Save(email Email) (string, error) {
	filename := fmt.Sprintf("%d.txt", time.Now().UnixNano())
	path := filepath.Join(fs.Dir, filename)
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	content := fmt.Sprintf("From: %s\nTo: %s\n\n%s", email.From, email.To, email.Content)
	if _, err := f.WriteString(content); err != nil {
		return "", err
	}
	return path, nil
}
