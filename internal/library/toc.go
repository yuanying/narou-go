package library

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// TOC is the contents of a novel toc.yaml.
type TOC struct {
	Title     string     `yaml:"title"`
	Author    string     `yaml:"author"`
	TocURL    string     `yaml:"toc_url"`
	Story     string     `yaml:"story"`
	Subtitles []Subtitle `yaml:"subtitles"`
}

// Subtitle describes one episode in toc.yaml.
type Subtitle struct {
	Index        string `yaml:"index"`
	Href         string `yaml:"href"`
	Chapter      string `yaml:"chapter"`
	Subchapter   string `yaml:"subchapter"`
	Subtitle     string `yaml:"subtitle"`
	FileSubtitle string `yaml:"file_subtitle"`
	Subdate      string `yaml:"subdate"`
	Subupdate    string `yaml:"subupdate"`
	DownloadTime string `yaml:"download_time,omitempty"`
}

// LoadTOC reads a novel toc.yaml.
func LoadTOC(novelDir string) (*TOC, error) {
	path := filepath.Join(novelDir, "toc.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read toc: %w", err)
	}

	var toc TOC
	if err := yaml.Unmarshal(data, &toc); err != nil {
		return nil, fmt.Errorf("parse toc: %w", err)
	}

	return &toc, nil
}
