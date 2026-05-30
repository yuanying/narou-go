package downloader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/yuanying/narou-go/internal/model"
)

func TestDownloadImagesDedupesAndContinuesOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/webp")
		_, _ = w.Write([]byte("image"))
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.Wait = 0
	client.Retries = 0
	client.Client = &http.Client{Timeout: time.Second}

	images := DownloadImages(context.Background(), client, []model.Image{
		{URL: server.URL + "/image"},
		{URL: server.URL + "/image"},
		{URL: server.URL + "/missing"},
	}, t.TempDir())

	if len(images) != 3 {
		t.Fatalf("len(images) = %d, want 3", len(images))
	}
	if images[0].LocalPath == "" || images[0].MediaType != "image/webp" {
		t.Fatalf("first image = %#v", images[0])
	}
	if images[1].LocalPath != images[0].LocalPath {
		t.Fatalf("duplicate local path = %q, want %q", images[1].LocalPath, images[0].LocalPath)
	}
	if filepath.Ext(images[0].LocalPath) != ".webp" {
		t.Fatalf("ext = %q, want .webp", filepath.Ext(images[0].LocalPath))
	}
	if !images[2].Failed || images[2].Warning == "" {
		t.Fatalf("missing image = %#v", images[2])
	}
}
