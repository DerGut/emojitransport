package emoji

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Store struct {
	directory string
	catalog   *os.File
}

func NewStore(path string) (*Store, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("checking path: %w", err)
	}

	if !info.IsDir() {
		return nil, errors.New("path is not a directory")
	}

	const catalogName = "emoji.catalog"
	catalogPath := filepath.Join(path, catalogName)

	file, err := os.Create(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("create catalog file: %w", err)
	}

	return &Store{
		directory: path,
		catalog:   file,
	}, nil
}

func (s *Store) Close() error {
	return s.catalog.Close()
}

func (s *Store) Store(emoji SlackEmoji, body io.Reader) error {
	name, err := fileName(emoji.URL)
	if err != nil {
		return fmt.Errorf("compute file name: %w", err)
	}

	if err := s.writeToFile(name, body); err != nil {
		return fmt.Errorf("writing to file: %w", err)
	}

	if err := s.appendToCatalog(name, emoji); err != nil {
		return fmt.Errorf("appending to catalog: %w", err)
	}

	return nil
}

func (s *Store) writeToFile(fileName string, content io.Reader) error {
	path := filepath.Join(s.directory, fileName)

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	if _, err := io.Copy(file, content); err != nil {
		return fmt.Errorf("write to file: %w", err)
	}

	return nil
}

func (s *Store) appendToCatalog(fileName string, emoji SlackEmoji) error {
	entry := struct {
		FileName string     `json:"fileName"`
		Emoji    SlackEmoji `json:"emoji"`
	}{
		FileName: fileName,
		Emoji:    emoji,
	}

	b, err := json.Marshal(&entry)
	if err != nil {
		return fmt.Errorf("marshal catalog entry: %w", err)
	}

	if _, err := s.catalog.Write(b); err != nil {
		return fmt.Errorf("write catalog entry: %w", err)
	}

	if _, err := s.catalog.WriteString("\n"); err != nil {
		return fmt.Errorf("writing new line: %w", err)
	}

	return nil
}

func fileName(url string) (string, error) {
	hash := sha256.New()
	if _, err := hash.Write([]byte(url)); err != nil {
		return "", fmt.Errorf("write url: %w", err)
	}

	name := hash.Sum(nil)

	ext := filepath.Ext(url)

	return fmt.Sprintf("%x.%s", name, ext), nil
}
