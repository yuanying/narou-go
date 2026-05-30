package downloader

import "testing"

func TestMediaTypeToExtension(t *testing.T) {
	tests := map[string]string{
		"image/jpeg":           ".jpg",
		"image/png; charset=x": ".png",
		"image/gif":            ".gif",
		"image/webp":           ".webp",
	}

	for input, want := range tests {
		got, err := MediaTypeToExtension(input)
		if err != nil {
			t.Fatalf("MediaTypeToExtension(%q) error = %v", input, err)
		}
		if got != want {
			t.Fatalf("MediaTypeToExtension(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMediaTypeToExtensionRejectsUnsupportedType(t *testing.T) {
	if _, err := MediaTypeToExtension("text/html"); err == nil {
		t.Fatal("MediaTypeToExtension() error = nil, want error")
	}
}
