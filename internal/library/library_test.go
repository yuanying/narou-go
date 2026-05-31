package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDatabase(t *testing.T) {
	db, err := LoadDatabase(filepath.Join("testdata", "library"))
	if err != nil {
		t.Fatalf("LoadDatabase() error = %v", err)
	}

	entry := db[0]
	if entry.Title != "サンプル小説" {
		t.Fatalf("Title = %q, want %q", entry.Title, "サンプル小説")
	}
	if entry.SiteName != "小説家になろう" {
		t.Fatalf("SiteName = %q, want %q", entry.SiteName, "小説家になろう")
	}
	if len(entry.Tags) != 1 || entry.Tags[0] != "sample" {
		t.Fatalf("Tags = %#v, want [sample]", entry.Tags)
	}
}

func TestFindByNcode(t *testing.T) {
	db, err := LoadDatabase(filepath.Join("testdata", "library"))
	if err != nil {
		t.Fatalf("LoadDatabase() error = %v", err)
	}

	entry, err := FindByNcode(db, "N1231ID")
	if err != nil {
		t.Fatalf("FindByNcode() error = %v", err)
	}
	if entry.ID != 0 {
		t.Fatalf("ID = %d, want 0", entry.ID)
	}

	if _, err := FindByNcode(db, "n0000aa"); err == nil {
		t.Fatal("FindByNcode() error = nil, want error")
	}
}

func TestNovelDir(t *testing.T) {
	entry := NovelEntry{SiteName: "小説家になろう", FileTitle: "n1231id サンプル小説"}
	got := NovelDir("library", entry)
	want := filepath.Join("library", "小説データ", "小説家になろう", "n1231id サンプル小説")
	if got != want {
		t.Fatalf("NovelDir() = %q, want %q", got, want)
	}
}

func TestLoadTOC(t *testing.T) {
	novelDir := filepath.Join("testdata", "library", "小説データ", "小説家になろう", "n1231id サンプル小説")
	toc, err := LoadTOC(novelDir)
	if err != nil {
		t.Fatalf("LoadTOC() error = %v", err)
	}

	if toc.Title != "サンプル小説" {
		t.Fatalf("Title = %q, want %q", toc.Title, "サンプル小説")
	}
	if len(toc.Subtitles) != 1 {
		t.Fatalf("len(Subtitles) = %d, want 1", len(toc.Subtitles))
	}
	if toc.Subtitles[0].Index != "1" {
		t.Fatalf("Index = %q, want %q", toc.Subtitles[0].Index, "1")
	}
}

func TestLoadTOCKeepsKakuyomuEpisodeID(t *testing.T) {
	novelDir := filepath.Join("testdata", "library", "小説データ", "カクヨム", "1177354054882961557 カクヨム小説")
	toc, err := LoadTOC(novelDir)
	if err != nil {
		t.Fatalf("LoadTOC() error = %v", err)
	}

	if got := toc.Subtitles[0].Index; got != "1177354054882961573" {
		t.Fatalf("Index = %q, want %q", got, "1177354054882961573")
	}
}

func TestLoadSection(t *testing.T) {
	novelDir := filepath.Join("testdata", "library", "小説データ", "小説家になろう", "n1231id サンプル小説")
	section, err := LoadSection(novelDir, Subtitle{Index: "1", FileSubtitle: "第一話"})
	if err != nil {
		t.Fatalf("LoadSection() error = %v", err)
	}

	if section.Element.DataType != "html" {
		t.Fatalf("DataType = %q, want %q", section.Element.DataType, "html")
	}
	if section.Element.Body == "" || section.Element.Introduction == "" || section.Element.Postscript == "" {
		t.Fatalf("Element = %#v, want introduction/body/postscript", section.Element)
	}
}

func TestLoadDownloadedNovelRestoresBody(t *testing.T) {
	root := filepath.Join("testdata", "library")
	db, err := LoadDatabase(root)
	if err != nil {
		t.Fatalf("LoadDatabase() error = %v", err)
	}
	entry := db[0]

	novel, err := LoadDownloadedNovel(root, entry)
	if err != nil {
		t.Fatalf("LoadDownloadedNovel() error = %v", err)
	}

	if novel.ID != "n1231id" || novel.Title != "サンプル小説" {
		t.Fatalf("novel = %#v", novel)
	}
	if len(novel.Episodes) != 1 {
		t.Fatalf("len(Episodes) = %d, want 1", len(novel.Episodes))
	}
	episode := novel.Episodes[0]
	if episode.Body == "" || episode.BodyHash == "" {
		t.Fatalf("episode was not restored: %#v", episode)
	}
}

func TestLoadDownloadedNovelKeepsEpisodeWhenSectionMissing(t *testing.T) {
	root := copyTestLibrary(t)
	db, err := LoadDatabase(root)
	if err != nil {
		t.Fatalf("LoadDatabase() error = %v", err)
	}
	entry := db[0]
	sectionPath := filepath.Join(root, "小説データ", "小説家になろう", "n1231id サンプル小説", "本文", "1 第一話.yaml")
	if err := os.Remove(sectionPath); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	novel, err := LoadDownloadedNovel(root, entry)
	if err != nil {
		t.Fatalf("LoadDownloadedNovel() error = %v", err)
	}
	if len(novel.Episodes) != 1 {
		t.Fatalf("len(Episodes) = %d, want 1", len(novel.Episodes))
	}
	if novel.Episodes[0].BodyHash != "" || novel.Episodes[0].Body != "" {
		t.Fatalf("missing section episode = %#v, want empty body", novel.Episodes[0])
	}
}

func TestListSectionFiles(t *testing.T) {
	novelDir := filepath.Join("testdata", "library", "小説データ", "小説家になろう", "n1231id サンプル小説")
	files, err := ListSectionFiles(novelDir)
	if err != nil {
		t.Fatalf("ListSectionFiles() error = %v", err)
	}

	want := []string{filepath.Join(novelDir, "本文", "1 第一話.yaml")}
	if len(files) != len(want) || files[0] != want[0] {
		t.Fatalf("ListSectionFiles() = %#v, want %#v", files, want)
	}
}

func copyTestLibrary(t *testing.T) string {
	t.Helper()

	src, err := filepath.Abs(filepath.Join("testdata", "library"))
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}
	dst := t.TempDir()
	if err := filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	}); err != nil {
		t.Fatalf("copy test library: %v", err)
	}

	return dst
}
