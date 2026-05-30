package kakuyomu

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchAndNormalize(t *testing.T) {
	d := New(nil)
	inputs := []string{
		"https://kakuyomu.jp/works/1177354054880241118",
		"https://kakuyomu.jp/works/1177354054880241118/episodes/1177354054880241182",
	}
	for _, input := range inputs {
		if !d.Match(input) {
			t.Fatalf("Match(%q) = false", input)
		}
		got, err := d.Normalize(input)
		if err != nil {
			t.Fatalf("Normalize(%q) error = %v", input, err)
		}
		if got != "kakuyomu-1177354054880241118" {
			t.Fatalf("Normalize(%q) = %q", input, got)
		}
	}
}

func TestParseWork(t *testing.T) {
	novel, err := ParseWork(readFixture(t, "work.html"))
	if err != nil {
		t.Fatalf("ParseWork() error = %v", err)
	}

	if novel.ID != "kakuyomu-1177354054880241118" || novel.Title != "カクヨムタイトル" || novel.Author != "作者名" {
		t.Fatalf("novel metadata = %#v", novel)
	}
	if len(novel.Chapters) != 1 || novel.Chapters[0].Title != "第一章" {
		t.Fatalf("chapters = %#v", novel.Chapters)
	}
	if len(novel.Episodes) != 2 {
		t.Fatalf("len(Episodes) = %d, want 2", len(novel.Episodes))
	}
	if novel.Episodes[0].Chapter != "第一章" || novel.Episodes[0].URL != "https://kakuyomu.jp/works/1177354054880241118/episodes/1177354054880241182" {
		t.Fatalf("first episode = %#v", novel.Episodes[0])
	}
}

func TestParseEpisode(t *testing.T) {
	body, images, err := ParseEpisode(readFixture(t, "episode.html"), "https://kakuyomu.jp/works/1177354054880241118/episodes/1177354054880241182")
	if err != nil {
		t.Fatalf("ParseEpisode() error = %v", err)
	}
	if body == "" {
		t.Fatal("body is empty")
	}
	if len(images) != 0 {
		t.Fatalf("images = %#v, want none", images)
	}
}

func TestParseEpisodeExtractsUniqueImages(t *testing.T) {
	_, images, err := ParseEpisode(readFixture(t, "episode_with_image.html"), "https://kakuyomu.jp/works/1177354054880241118/episodes/1177354054880241182")
	if err != nil {
		t.Fatalf("ParseEpisode() error = %v", err)
	}
	if len(images) != 1 || images[0] != "https://kakuyomu.jp/images/sample.webp" {
		t.Fatalf("images = %#v", images)
	}
}

func readFixture(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	return string(data)
}
