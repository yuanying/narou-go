package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yuanying/narou-go/internal/model"
	"gopkg.in/yaml.v3"
)

const NovelFileName = "novel.yaml"

// NovelDir returns the Go downloader storage directory for a novel.
func NovelDir(root string, novelID string) string {
	return filepath.Join(root, novelID)
}

// SaveNovel writes a downloaded novel to data/{id}/novel.yaml.
func SaveNovel(root string, novel *model.Novel) error {
	dir := NovelDir(root, novel.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create novel dir: %w", err)
	}

	data, err := yaml.Marshal(novel)
	if err != nil {
		return fmt.Errorf("marshal novel: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, NovelFileName), data, 0o644); err != nil {
		return fmt.Errorf("write novel: %w", err)
	}

	return nil
}

// LoadNovel reads data/{id}/novel.yaml.
func LoadNovel(root string, novelID string) (*model.Novel, error) {
	data, err := os.ReadFile(filepath.Join(NovelDir(root, novelID), NovelFileName))
	if err != nil {
		return nil, fmt.Errorf("read novel: %w", err)
	}

	var novel model.Novel
	if err := yaml.Unmarshal(data, &novel); err != nil {
		return nil, fmt.Errorf("parse novel: %w", err)
	}

	return &novel, nil
}
