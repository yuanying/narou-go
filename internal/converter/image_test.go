package converter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImageRegistryResolvesMiteminURL(t *testing.T) {
	novelDir := t.TempDir()
	writeTestImage(t, filepath.Join(novelDir, "挿絵", "i514881.jpg"))

	registry := NewImageRegistry(novelDir)
	got, err := registry.Resolve("https://12345.mitemin.net/userpageimage/viewimage/icode/514881/")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if got != "../images/i514881.jpg" {
		t.Fatalf("Resolve() = %q, want %q", got, "../images/i514881.jpg")
	}
	assets := registry.Assets()
	if len(assets) != 1 {
		t.Fatalf("len(Assets()) = %d, want 1", len(assets))
	}
	if assets[0].SourcePath != filepath.Join(novelDir, "挿絵", "i514881.jpg") {
		t.Fatalf("SourcePath = %q", assets[0].SourcePath)
	}
	if assets[0].Href != "images/i514881.jpg" {
		t.Fatalf("Href = %q", assets[0].Href)
	}
	if assets[0].MediaType != "image/jpeg" {
		t.Fatalf("MediaType = %q", assets[0].MediaType)
	}
}

func TestImageRegistryResolvesLocalPath(t *testing.T) {
	novelDir := t.TempDir()
	writeTestImage(t, filepath.Join(novelDir, "挿絵", "local.png"))

	registry := NewImageRegistry(novelDir)
	got, err := registry.Resolve("挿絵/local.png")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if got != "../images/local.png" {
		t.Fatalf("Resolve() = %q, want %q", got, "../images/local.png")
	}
	assets := registry.Assets()
	if assets[0].MediaType != "image/png" {
		t.Fatalf("MediaType = %q", assets[0].MediaType)
	}
}

func TestImageRegistryReturnsErrorForMissingImage(t *testing.T) {
	registry := NewImageRegistry(t.TempDir())
	if _, err := registry.Resolve("https://12345.mitemin.net/userpageimage/viewimage/icode/999999/"); err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}
}

func writeTestImage(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("image"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
