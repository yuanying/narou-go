package library

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Section is one episode YAML from 本文/.
type Section struct {
	Index        string  `yaml:"index"`
	Href         string  `yaml:"href"`
	Chapter      string  `yaml:"chapter"`
	Subchapter   string  `yaml:"subchapter"`
	Subtitle     string  `yaml:"subtitle"`
	FileSubtitle string  `yaml:"file_subtitle"`
	Subdate      string  `yaml:"subdate"`
	Subupdate    string  `yaml:"subupdate"`
	DownloadTime string  `yaml:"download_time,omitempty"`
	Element      Element `yaml:"element"`
}

// Element contains the HTML fragments stored by narou.rb.
type Element struct {
	DataType     string `yaml:"data_type"`
	Introduction string `yaml:"introduction"`
	Body         string `yaml:"body"`
	Postscript   string `yaml:"postscript"`
}

// LoadSection reads 本文/{index} {file_subtitle}.yaml.
func LoadSection(novelDir string, subtitle Subtitle) (*Section, error) {
	name := subtitle.Index + " " + subtitle.FileSubtitle + ".yaml"
	path := filepath.Join(novelDir, "本文", name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read section: %w", err)
	}

	var section Section
	if err := yaml.Unmarshal(data, &section); err != nil {
		return nil, fmt.Errorf("parse section: %w", err)
	}

	return &section, nil
}

// ListSectionFiles returns YAML files directly under 本文/ in stable order.
func ListSectionFiles(novelDir string) ([]string, error) {
	bodyDir := filepath.Join(novelDir, "本文")
	entries, err := os.ReadDir(bodyDir)
	if err != nil {
		return nil, fmt.Errorf("read section dir: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "._") || !strings.HasSuffix(name, ".yaml") {
			continue
		}
		files = append(files, filepath.Join(bodyDir, name))
	}

	sort.Strings(files)
	return files, nil
}
