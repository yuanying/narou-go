package epub

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestBuildCreatesMinimalEPUB3(t *testing.T) {
	var buf bytes.Buffer
	err := Build(&buf, Book{
		Title:    "サンプル小説",
		Author:   "テスト著者",
		Language: "ja",
		Sections: []Section{
			{Title: "第一話", Content: `<p>本文</p>`},
			{Title: "第二話", Content: `<p>続き</p>`},
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}

	if reader.File[0].Name != "mimetype" {
		t.Fatalf("first entry = %q, want mimetype", reader.File[0].Name)
	}
	if reader.File[0].Method != zip.Store {
		t.Fatalf("mimetype method = %d, want %d", reader.File[0].Method, zip.Store)
	}
	if got := readZipFile(t, reader, "mimetype"); got != "application/epub+zip" {
		t.Fatalf("mimetype = %q", got)
	}

	required := []string{
		"META-INF/container.xml",
		"OEBPS/package.opf",
		"OEBPS/nav.xhtml",
		"OEBPS/style/vertical.css",
		"OEBPS/text/p001.xhtml",
		"OEBPS/text/p002.xhtml",
	}
	for _, name := range required {
		if !hasZipFile(reader, name) {
			t.Fatalf("missing EPUB entry %q", name)
		}
	}
}

func TestBuildWritesMetadataNavigationAndContent(t *testing.T) {
	var buf bytes.Buffer
	err := Build(&buf, Book{
		Title:  "A&B",
		Author: "著者",
		Sections: []Section{
			{Title: "第一話", Content: `<p>本文<strong>太字</strong></p>`},
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}

	opf := readZipFile(t, reader, "OEBPS/package.opf")
	for _, want := range []string{
		`<dc:title>A&amp;B</dc:title>`,
		`<dc:creator>著者</dc:creator>`,
		`<dc:language>ja</dc:language>`,
		`<meta name="primary-writing-mode" content="vertical-rl"></meta>`,
		`<item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"></item>`,
		`<spine page-progression-direction="rtl">`,
		`<itemref idref="p001"></itemref>`,
	} {
		if !strings.Contains(opf, want) {
			t.Fatalf("package.opf missing %q:\n%s", want, opf)
		}
	}

	nav := readZipFile(t, reader, "OEBPS/nav.xhtml")
	if !strings.Contains(nav, `<a href="text/p001.xhtml">第一話</a>`) {
		t.Fatalf("nav.xhtml missing first section link:\n%s", nav)
	}

	content := readZipFile(t, reader, "OEBPS/text/p001.xhtml")
	for _, want := range []string{
		`<title>第一話</title>`,
		`<link rel="stylesheet" type="text/css" href="../style/vertical.css" />`,
		`<body><h1>第一話</h1><p>本文<strong>太字</strong></p></body>`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("p001.xhtml missing %q:\n%s", want, content)
		}
	}
}

func readZipFile(t *testing.T, reader *zip.Reader, name string) string {
	t.Helper()

	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		defer rc.Close()

		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		return buf.String()
	}

	t.Fatalf("missing %s", name)
	return ""
}

func hasZipFile(reader *zip.Reader, name string) bool {
	for _, file := range reader.File {
		if file.Name == name {
			return true
		}
	}

	return false
}
