package epub

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildEmbedsImagesAndMarksCover(t *testing.T) {
	imagePath := filepath.Join(t.TempDir(), "i514881.jpg")
	if err := os.WriteFile(imagePath, []byte("image-data"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var buf bytes.Buffer
	err := Build(&buf, Book{
		Title:  "画像付き",
		Author: "著者",
		Sections: []Section{
			{Title: "第一話", Content: `<p><img src="../images/i514881.jpg" /></p>`},
		},
		Images: []Image{
			{Href: "images/i514881.jpg", SourcePath: imagePath, MediaType: "image/jpeg"},
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}

	if got := readZipFile(t, reader, "OEBPS/images/i514881.jpg"); got != "image-data" {
		t.Fatalf("embedded image = %q", got)
	}

	opf := readZipFile(t, reader, "OEBPS/package.opf")
	for _, want := range []string{
		`<item id="image-001" href="images/i514881.jpg" media-type="image/jpeg" properties="cover-image"></item>`,
	} {
		if !strings.Contains(opf, want) {
			t.Fatalf("package.opf missing %q:\n%s", want, opf)
		}
	}
}
