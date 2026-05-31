package epub

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildWithCoverImageGeneratesCoverPage(t *testing.T) {
	imagePath := filepath.Join(t.TempDir(), "cover.jpg")
	if err := os.WriteFile(imagePath, []byte("cover-data"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var buf bytes.Buffer
	err := Build(&buf, Book{
		Title:      "表紙付き小説",
		Author:     "著者",
		CoverImage: "images/cover.jpg",
		Sections: []Section{
			{Title: "第一話", Content: `<p>本文</p>`},
		},
		Images: []Image{
			{Href: "images/cover.jpg", SourcePath: imagePath, MediaType: "image/jpeg"},
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}

	if !hasZipFile(reader, "OEBPS/cover.xhtml") {
		t.Fatal("missing OEBPS/cover.xhtml")
	}

	cover := readZipFile(t, reader, "OEBPS/cover.xhtml")
	if !strings.Contains(cover, `src="images/cover.jpg"`) {
		t.Fatalf("cover.xhtml missing cover image src:\n%s", cover)
	}
	if !strings.Contains(cover, `class="cover"`) {
		t.Fatalf("cover.xhtml missing cover class:\n%s", cover)
	}
	if !strings.Contains(cover, "表紙付き小説") {
		t.Fatalf("cover.xhtml missing title:\n%s", cover)
	}
	if !strings.Contains(cover, "著者") {
		t.Fatalf("cover.xhtml missing author:\n%s", cover)
	}

	opf := readZipFile(t, reader, "OEBPS/package.opf")
	if !strings.Contains(opf, `id="cover-page" href="cover.xhtml"`) {
		t.Fatalf("package.opf missing cover.xhtml manifest item:\n%s", opf)
	}
	// cover page must be first in spine
	coverIdx := strings.Index(opf, `idref="cover-page"`)
	p001Idx := strings.Index(opf, `idref="p001"`)
	if coverIdx < 0 {
		t.Fatalf("package.opf missing cover-page spine item:\n%s", opf)
	}
	if coverIdx > p001Idx {
		t.Fatalf("cover-page spine item must come before p001:\n%s", opf)
	}
}

func TestBuildWithoutCoverImageStillHasTitlePage(t *testing.T) {
	var buf bytes.Buffer
	err := Build(&buf, Book{
		Title:  "表紙なし",
		Author: "著者",
		Sections: []Section{
			{Title: "第一話", Content: `<p>本文</p>`},
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}

	if !hasZipFile(reader, "OEBPS/cover.xhtml") {
		t.Fatal("OEBPS/cover.xhtml must exist even without a cover image")
	}

	cover := readZipFile(t, reader, "OEBPS/cover.xhtml")
	if !strings.Contains(cover, "表紙なし") {
		t.Fatalf("cover.xhtml missing title:\n%s", cover)
	}
	if !strings.Contains(cover, "著者") {
		t.Fatalf("cover.xhtml missing author:\n%s", cover)
	}
	// no img tag when there is no cover image
	if strings.Contains(cover, "<img") {
		t.Fatalf("cover.xhtml must not contain img when CoverImage is empty:\n%s", cover)
	}
}

func TestBuildChapterPageHasCenteredTitle(t *testing.T) {
	var buf bytes.Buffer
	err := Build(&buf, Book{
		Title:  "章表紙テスト",
		Author: "著者",
		Sections: []Section{
			{Title: "第一章", ChapterPage: true},
			{Title: "第一話", Content: `<p>本文</p>`},
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}

	chapterPage := readZipFile(t, reader, "OEBPS/text/p001.xhtml")
	if !strings.Contains(chapterPage, `class="chapter-page"`) {
		t.Fatalf("chapter page missing chapter-page class:\n%s", chapterPage)
	}
	if !strings.Contains(chapterPage, `第一章`) {
		t.Fatalf("chapter page missing chapter title:\n%s", chapterPage)
	}

	// Regular sections must not have chapter-page class
	episodePage := readZipFile(t, reader, "OEBPS/text/p002.xhtml")
	if strings.Contains(episodePage, `class="chapter-page"`) {
		t.Fatalf("regular section must not have chapter-page class:\n%s", episodePage)
	}
}
