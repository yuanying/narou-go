package main

import (
	"archive/zip"
	"bytes"
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

func TestRunConvert(t *testing.T) {
	output := filepath.Join(t.TempDir(), "sample.epub")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"convert", "n1231id", "--library", fixtureLibraryPath(t), "--output", output}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run() code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), output) {
		t.Fatalf("convert output = %q, want output path", stdout.String())
	}

	reader, err := zip.OpenReader(output)
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

func TestRunConvertRequiresNcode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"convert", "--library", fixtureLibraryPath(t)}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("run() code = 0, stderr = %s", stderr.String())
	}
}

func fixtureLibraryPath(t *testing.T) string {
	t.Helper()

	return filepath.Join("..", "..", "internal", "library", "testdata", "library")
}

func zipHasFile(reader *zip.Reader, name string) bool {
	for _, file := range reader.File {
		if file.Name == name {
			return true
		}
	}

	return false
}
