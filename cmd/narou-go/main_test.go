package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunList(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"list", "--library", fixtureLibraryPath(t)}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run() code = %d, stderr = %s", code, stderr.String())
	}

	got := stdout.String()
	for _, want := range []string{"0", "n1231id", "サンプル小説", "テスト著者"} {
		if !strings.Contains(got, want) {
			t.Fatalf("list output missing %q:\n%s", want, got)
		}
	}
}

func TestRunListUsesCurrentLibraryRoot(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root, err := filepath.Abs(fixtureLibraryPath(t))
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}
	t.Chdir(root)

	code := run([]string{"list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run() code = %d, stderr = %s", code, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "サンプル小説") {
		t.Fatalf("list output = %q, want sample novel", got)
	}
}

func TestRunConvert(t *testing.T) {
	libraryPath := copyFixtureLibrary(t)
	output := filepath.Join(t.TempDir(), "sample.epub")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"convert", "n1231id", "--library", libraryPath, "--output", output}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run() code = %d, stderr = %s", code, stderr.String())
	}
	converted := filepath.Join(libraryPath, "小説データ", "小説家になろう", "n1231id サンプル小説", "sample.epub")
	if !strings.Contains(stdout.String(), converted) {
		t.Fatalf("convert output = %q, want output path %q", stdout.String(), converted)
	}

	reader, err := zip.OpenReader(converted)
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}
	defer reader.Close()

	if !zipHasFile(&reader.Reader, "OEBPS/package.opf") {
		t.Fatal("converted EPUB missing OEBPS/package.opf")
	}
	if !zipHasFile(&reader.Reader, "OEBPS/text/p001.xhtml") {
		t.Fatal("converted EPUB missing OEBPS/text/p001.xhtml")
	}
}

func TestRunConvertDefaultOutputUsesNovelDirectory(t *testing.T) {
	libraryPath := copyFixtureLibrary(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"convert", "n1231id", "--library", libraryPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run() code = %d, stderr = %s", code, stderr.String())
	}

	converted := filepath.Join(libraryPath, "小説データ", "小説家になろう", "n1231id サンプル小説", "[テスト著者] サンプル小説.epub")
	if !strings.Contains(stdout.String(), converted) {
		t.Fatalf("convert output = %q, want output path %q", stdout.String(), converted)
	}
	if _, err := os.Stat(converted); err != nil {
		t.Fatalf("converted file missing: %v", err)
	}
}

func TestRunConvertRequiresNcode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"convert", "--library", fixtureLibraryPath(t)}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("run() code = 0, stderr = %s", stderr.String())
	}
}

func TestParseConvertOptionsKindle(t *testing.T) {
	got, err := parseConvertOptions([]string{"n1231id", "--kindle"})
	if err != nil {
		t.Fatalf("parseConvertOptions() error = %v", err)
	}
	if got.ncode != "n1231id" || !got.kindle {
		t.Fatalf("parseConvertOptions() = %#v", got)
	}
}

func TestParseWebOptions(t *testing.T) {
	got, err := parseWebOptions("download", []string{"--epub", "--library", "library-dir", "n9669bk"})
	if err != nil {
		t.Fatalf("parseWebOptions() error = %v", err)
	}
	if got.target != "n9669bk" || got.libraryPath != "library-dir" || !got.epub {
		t.Fatalf("parseWebOptions() = %#v", got)
	}
}

func TestParseWebOptionsRejectsData(t *testing.T) {
	if _, err := parseWebOptions("download", []string{"--data", "data-dir", "n9669bk"}); err == nil {
		t.Fatal("parseWebOptions() error = nil, want unknown option")
	}
}

func TestParseWebOptionsKindleImpliesEPUB(t *testing.T) {
	got, err := parseWebOptions("download", []string{"--kindle", "n9669bk"})
	if err != nil {
		t.Fatalf("parseWebOptions() error = %v", err)
	}
	if !got.kindle || !got.epub {
		t.Fatalf("parseWebOptions() = %#v, want kindle and epub", got)
	}
}

func TestCreateKindleRunsAphrael(t *testing.T) {
	binDir := t.TempDir()
	script := filepath.Join(binDir, "aphrael")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncp \"$1\" \"$2\"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	epubPath := filepath.Join(t.TempDir(), "sample.epub")
	if err := os.WriteFile(epubPath, []byte("epub"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := createKindle(epubPath)
	if err != nil {
		t.Fatalf("createKindle() error = %v", err)
	}
	if got != filepath.Join(filepath.Dir(epubPath), "sample.mobi") {
		t.Fatalf("createKindle() = %q", got)
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("kindle output was not created: %v", err)
	}
}

func TestSelectDownloader(t *testing.T) {
	if _, err := selectDownloader("n9669bk"); err != nil {
		t.Fatalf("selectDownloader(syosetu) error = %v", err)
	}
	if _, err := selectDownloader("https://kakuyomu.jp/works/1177354054880241118"); err != nil {
		t.Fatalf("selectDownloader(kakuyomu) error = %v", err)
	}
	if _, err := selectDownloader("https://example.com/"); err == nil {
		t.Fatal("selectDownloader() error = nil, want error")
	}
}

func fixtureLibraryPath(t *testing.T) string {
	t.Helper()

	return filepath.Join("..", "..", "internal", "library", "testdata", "library")
}

func copyFixtureLibrary(t *testing.T) string {
	t.Helper()

	src, err := filepath.Abs(fixtureLibraryPath(t))
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
		t.Fatalf("copy fixture library: %v", err)
	}

	return dst
}

func zipHasFile(reader *zip.Reader, name string) bool {
	for _, file := range reader.File {
		if file.Name == name {
			return true
		}
	}

	return false
}
