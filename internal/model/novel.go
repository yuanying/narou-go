package model

// Novel is the site-neutral representation produced by web downloaders.
type Novel struct {
	ID        string    `yaml:"id"`
	Site      string    `yaml:"site"`
	SourceURL string    `yaml:"source_url"`
	Title     string    `yaml:"title"`
	Author    string    `yaml:"author"`
	Story     string    `yaml:"story"`
	NovelType int       `yaml:"novel_type"`
	End       bool      `yaml:"end"`
	Chapters  []Chapter `yaml:"chapters"`
	Episodes  []Episode `yaml:"episodes"`
	Images    []Image   `yaml:"images"`
}

// Chapter preserves chapter headings found in a table of contents.
type Chapter struct {
	Title string `yaml:"title"`
	Level int    `yaml:"level"`
}

// Episode is one downloaded section.
type Episode struct {
	ID           string `yaml:"id"`
	Title        string `yaml:"title"`
	Chapter      string `yaml:"chapter"`
	Subchapter   string `yaml:"subchapter"`
	URL          string `yaml:"url"`
	PublishedAt  string `yaml:"published_at"`
	UpdatedAt    string `yaml:"updated_at"`
	Preface      string `yaml:"preface"`
	Body         string `yaml:"body"`
	Afterword    string `yaml:"afterword"`
	BodyHash     string `yaml:"body_hash"`
	DownloadedAt string `yaml:"downloaded_at"`
}

// Image is one downloaded image referenced by the novel body.
type Image struct {
	URL       string `yaml:"url"`
	LocalPath string `yaml:"local_path"`
	MediaType string `yaml:"media_type"`
	Failed    bool   `yaml:"failed,omitempty"`
	Warning   string `yaml:"warning,omitempty"`
}
