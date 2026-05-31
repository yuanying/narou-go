package downloader

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuanying/narou-go/internal/model"
)

var miteminImagePathPattern = regexp.MustCompile(`/icode/([0-9]+)/?`)

// DownloadImages downloads novel images into imageDir. Failures are warnings on the image.
func DownloadImages(ctx context.Context, client *HTTPClient, images []model.Image, imageDir string) []model.Image {
	if client == nil {
		client = NewHTTPClient()
	}
	seen := map[string]string{}
	results := make([]model.Image, 0, len(images))
	for _, image := range images {
		if image.URL == "" {
			continue
		}
		if existing, ok := seen[image.URL]; ok {
			image.LocalPath = existing
			results = append(results, image)
			continue
		}
		body, contentType, err := client.GetBytes(ctx, image.URL)
		if err != nil {
			image.Failed = true
			image.Warning = err.Error()
			results = append(results, image)
			continue
		}
		ext, err := imageExtension(image.URL, contentType)
		if err != nil {
			image.Failed = true
			image.Warning = err.Error()
			results = append(results, image)
			continue
		}
		name := imageFileName(image.URL, ext)
		if err := os.MkdirAll(imageDir, 0o755); err != nil {
			image.Failed = true
			image.Warning = err.Error()
			results = append(results, image)
			continue
		}
		path := filepath.Join(imageDir, name)
		if err := os.WriteFile(path, body, 0o644); err != nil {
			image.Failed = true
			image.Warning = err.Error()
			results = append(results, image)
			continue
		}
		image.LocalPath = path
		image.MediaType = mediaTypeOrDefault(contentType, path)
		seen[image.URL] = path
		results = append(results, image)
	}

	return results
}

func imageExtension(rawURL string, contentType string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err == nil {
		if ext := filepath.Ext(parsed.Path); ext != "" {
			switch strings.ToLower(ext) {
			case ".jpg", ".jpeg", ".png", ".gif", ".webp":
				return ext, nil
			}
		}
	}

	return MediaTypeToExtension(contentType)
}

func imageFileName(rawURL string, ext string) string {
	parsed, err := url.Parse(rawURL)
	if err == nil {
		if strings.HasSuffix(parsed.Host, ".mitemin.net") {
			matches := miteminImagePathPattern.FindStringSubmatch(parsed.Path)
			if len(matches) == 2 {
				return "i" + matches[1] + ext
			}
		}
		base := filepath.Base(parsed.Path)
		if base != "." && base != "/" && base != "" && filepath.Ext(base) != "" {
			return base
		}
	}
	hash := sha256.Sum256([]byte(rawURL))

	return fmt.Sprintf("%x%s", hash[:8], ext)
}

func mediaTypeOrDefault(contentType string, path string) string {
	mediaType := strings.Split(contentType, ";")[0]
	mediaType = strings.TrimSpace(mediaType)
	if mediaType != "" {
		return mediaType
	}

	return MediaTypeFromPath(path)
}
