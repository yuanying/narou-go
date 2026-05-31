package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRootUsesExplicitPath(t *testing.T) {
	root := t.TempDir()
	got, err := ResolveRoot(root)
	if err != nil {
		t.Fatalf("ResolveRoot() error = %v", err)
	}
	if got != root {
		t.Fatalf("ResolveRoot() = %q, want %q", got, root)
	}
}

func TestResolveRootFindsNarouDirectoryInParents(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".narou"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	child := filepath.Join(root, "小説データ", "小説家になろう")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	t.Chdir(child)

	got, err := ResolveRoot("")
	if err != nil {
		t.Fatalf("ResolveRoot() error = %v", err)
	}
	if got != root {
		t.Fatalf("ResolveRoot() = %q, want %q", got, root)
	}
}

func TestResolveRootFallsBackToCurrentDirectory(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	got, err := ResolveRoot("")
	if err != nil {
		t.Fatalf("ResolveRoot() error = %v", err)
	}
	if got != root {
		t.Fatalf("ResolveRoot() = %q, want %q", got, root)
	}
}
