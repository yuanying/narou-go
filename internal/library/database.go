package library

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// NovelEntry is one entry from library/.narou/database.yaml.
type NovelEntry struct {
	ID        int      `yaml:"id"`
	Author    string   `yaml:"author"`
	Title     string   `yaml:"title"`
	FileTitle string   `yaml:"file_title"`
	TocURL    string   `yaml:"toc_url"`
	SiteName  string   `yaml:"sitename"`
	NovelType int      `yaml:"novel_type"`
	End       bool     `yaml:"end"`
	Tags      []string `yaml:"tags"`
}

// LoadDatabase reads library/.narou/database.yaml.
func LoadDatabase(libraryPath string) (map[int]NovelEntry, error) {
	path := filepath.Join(libraryPath, ".narou", "database.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read database: %w", err)
	}

	db := make(map[int]NovelEntry)
	if err := yaml.Unmarshal(data, &db); err != nil {
		return nil, fmt.Errorf("parse database: %w", err)
	}

	return db, nil
}

// FindByNcode finds a syosetu novel by ncode using toc_url or file_title.
func FindByNcode(db map[int]NovelEntry, ncode string) (*NovelEntry, error) {
	normalized := strings.ToLower(strings.TrimSpace(ncode))
	if normalized == "" {
		return nil, fmt.Errorf("ncode is empty")
	}

	for _, entry := range db {
		fileTitle := strings.ToLower(entry.FileTitle)
		tocURL := strings.ToLower(entry.TocURL)
		if fileTitle == normalized || strings.HasPrefix(fileTitle, normalized+" ") || strings.Contains(tocURL, "/"+normalized+"/") {
			found := entry
			return &found, nil
		}
	}

	return nil, fmt.Errorf("ncode %q not found", ncode)
}
