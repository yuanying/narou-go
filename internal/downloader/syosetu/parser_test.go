package syosetu

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchAndNormalize(t *testing.T) {
	d := New(nil)
	inputs := []string{
		"n9669bk",
		"N9669BK",
		"https://ncode.syosetu.com/n9669bk/",
		"https://ncode.syosetu.com/n9669bk/1/",
	}
	for _, input := range inputs {
		if !d.Match(input) {
			t.Fatalf("Match(%q) = false", input)
		}
		got, err := d.Normalize(input)
		if err != nil {
			t.Fatalf("Normalize(%q) error = %v", input, err)
		}
		if got != "n9669bk" {
			t.Fatalf("Normalize(%q) = %q, want n9669bk", input, got)
		}
	}
}

func TestParseTOC(t *testing.T) {
	source := readFixture(t, "toc.html")
	novel, err := ParseTOC("n9669bk", source)
	if err != nil {
		t.Fatalf("ParseTOC() error = %v", err)
	}

	if novel.ID != "n9669bk" || novel.Title != "連載タイトル" || novel.Author != "作者名" {
		t.Fatalf("novel metadata = %#v", novel)
	}
	if len(novel.Episodes) != 2 {
		t.Fatalf("len(Episodes) = %d, want 2", len(novel.Episodes))
	}
	first := novel.Episodes[0]
	if first.ID != "1" || first.Chapter != "第一章" || first.Title != "第一話" || first.UpdatedAt != "2020/01/02 12:00" {
		t.Fatalf("first episode = %#v", first)
	}
}

func TestParseShortStory(t *testing.T) {
	novel, err := ParseShortStory("n1111aa", readFixture(t, "short.html"))
	if err != nil {
		t.Fatalf("ParseShortStory() error = %v", err)
	}
	if novel.NovelType != 2 || len(novel.Episodes) != 1 {
		t.Fatalf("short novel = %#v", novel)
	}
	if novel.Episodes[0].Body == "" || novel.Episodes[0].Title != "短編タイトル" {
		t.Fatalf("short episode = %#v", novel.Episodes[0])
	}
}

func TestParseEpisode(t *testing.T) {
	got, images, err := ParseEpisode(readFixture(t, "episode.html"), "https://ncode.syosetu.com/n9669bk/1/")
	if err != nil {
		t.Fatalf("ParseEpisode() error = %v", err)
	}
	if got.Preface == "" || got.Body == "" || got.Afterword == "" {
		t.Fatalf("episode = %#v", got)
	}
	if len(images) != 0 {
		t.Fatalf("images = %#v, want none", images)
	}
}

func TestParseEpisodeExtractsImages(t *testing.T) {
	got, images, err := ParseEpisode(readFixture(t, "episode_with_image.html"), "https://ncode.syosetu.com/n9669bk/1/")
	if err != nil {
		t.Fatalf("ParseEpisode() error = %v", err)
	}
	if got.Body == "" {
		t.Fatal("Body is empty")
	}
	if len(images) != 1 || images[0] != "https://12345.mitemin.net/userpageimage/viewimage/icode/514881/" {
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
