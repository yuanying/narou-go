package library

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yuanying/narou-go/internal/model"
	"gopkg.in/yaml.v3"
)

func TestSaveDownloadedNovelWritesNarouLibraryFiles(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	novel := &model.Novel{
		ID:        "n9669bk",
		Site:      "syosetu",
		SourceURL: "https://ncode.syosetu.com/n9669bk/",
		Title:     "タイトル",
		Author:    "作者",
		Story:     "あらすじ",
		NovelType: 1,
		Episodes: []model.Episode{
			{
				ID:           "1",
				Title:        "第一/話",
				Chapter:      "第一章",
				URL:          "https://ncode.syosetu.com/n9669bk/1/",
				PublishedAt:  "2020/01/01 00:00",
				UpdatedAt:    "2020/01/02 00:00",
				Preface:      "<p>前書き</p>",
				Body:         "<p id=\"L1\">本文</p>",
				Afterword:    "<p>後書き</p>",
				DownloadedAt: "2026-05-31T12:00:00Z",
			},
		},
	}

	entry, err := SaveDownloadedNovel(root, novel, now)
	if err != nil {
		t.Fatalf("SaveDownloadedNovel() error = %v", err)
	}
	if entry.ID != 0 || entry.FileTitle != "n9669bk タイトル" || entry.SiteName != "小説家になろう" {
		t.Fatalf("entry = %#v", entry)
	}

	db, err := LoadDatabase(root)
	if err != nil {
		t.Fatalf("LoadDatabase() error = %v", err)
	}
	if db[0].Title != "タイトル" || db[0].Author != "作者" {
		t.Fatalf("database entry = %#v", db[0])
	}

	novelDir := NovelDir(root, *entry)
	toc, err := LoadTOC(novelDir)
	if err != nil {
		t.Fatalf("LoadTOC() error = %v", err)
	}
	if toc.Title != "タイトル" || len(toc.Subtitles) != 1 {
		t.Fatalf("toc = %#v", toc)
	}
	if toc.Subtitles[0].FileSubtitle != "第一／話" {
		t.Fatalf("file_subtitle = %q", toc.Subtitles[0].FileSubtitle)
	}

	section, err := LoadSection(novelDir, toc.Subtitles[0])
	if err != nil {
		t.Fatalf("LoadSection() error = %v", err)
	}
	if section.Element.DataType != "html" || section.Element.Body != "<p id=\"L1\">本文</p>" {
		t.Fatalf("section = %#v", section)
	}
}

func TestSaveDownloadedNovelAppendsNextDatabaseID(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".narou"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	data := []byte("---\n2:\n  id: 2\n  title: 既存\n  author: 作者\n  file_title: n0000aa 既存\n  toc_url: https://ncode.syosetu.com/n0000aa/\n  sitename: 小説家になろう\n  novel_type: 1\n  end: false\n  tags: []\n")
	if err := os.WriteFile(filepath.Join(root, ".narou", "database.yaml"), data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	entry, err := SaveDownloadedNovel(root, minimalNovel("n1111aa", "追加"), time.Now())
	if err != nil {
		t.Fatalf("SaveDownloadedNovel() error = %v", err)
	}
	if entry.ID != 3 {
		t.Fatalf("entry.ID = %d, want 3", entry.ID)
	}
}

func TestUpdateDownloadedNovelKeepsIDAndFileTitle(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	entry, err := SaveDownloadedNovel(root, minimalNovel("n1111aa", "古いタイトル"), now)
	if err != nil {
		t.Fatalf("SaveDownloadedNovel() error = %v", err)
	}

	updated := minimalNovel("n1111aa", "新しいタイトル")
	updated.Episodes = append(updated.Episodes, model.Episode{ID: "2", Title: "第二話", URL: "https://ncode.syosetu.com/n1111aa/2/", Body: "<p>二話</p>"})
	got, err := UpdateDownloadedNovel(root, *entry, updated, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("UpdateDownloadedNovel() error = %v", err)
	}
	if got.ID != entry.ID || got.FileTitle != entry.FileTitle {
		t.Fatalf("updated entry = %#v, want id/file_title from %#v", got, entry)
	}

	toc, err := LoadTOC(NovelDir(root, *got))
	if err != nil {
		t.Fatalf("LoadTOC() error = %v", err)
	}
	if toc.Title != "新しいタイトル" || len(toc.Subtitles) != 2 {
		t.Fatalf("toc = %#v", toc)
	}
}

func TestDatabaseTimesAreYAMLTimestamps(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	if _, err := SaveDownloadedNovel(root, minimalNovel("n1111aa", "タイトル"), now); err != nil {
		t.Fatalf("SaveDownloadedNovel() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, ".narou", "database.yaml"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.Contains(string(data), `last_update: "`) {
		t.Fatalf("last_update was quoted:\n%s", data)
	}

	var db map[int]map[string]any
	if err := yaml.Unmarshal(data, &db); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := db[0]["last_update"].(time.Time); !ok {
		t.Fatalf("last_update = %#v, want time.Time", db[0]["last_update"])
	}

	if _, err := exec.LookPath("ruby"); err != nil {
		t.Skip("ruby not found")
	}
	cmd := exec.Command("ruby", "-ryaml", "-e", `db = YAML.unsafe_load_file(ARGV[0]); db[0]["last_update"].strftime("%y/%m/%d")`)
	cmd.Args = append(cmd.Args, filepath.Join(root, ".narou", "database.yaml"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ruby failed: %v: %s", err, output)
	}
}

func minimalNovel(id string, title string) *model.Novel {
	return &model.Novel{
		ID:        id,
		Site:      "syosetu",
		SourceURL: "https://ncode.syosetu.com/" + id + "/",
		Title:     title,
		Author:    "作者",
		NovelType: 1,
		Episodes: []model.Episode{
			{ID: "1", Title: "第一話", URL: "https://ncode.syosetu.com/" + id + "/1/", Body: "<p>本文</p>"},
		},
	}
}
