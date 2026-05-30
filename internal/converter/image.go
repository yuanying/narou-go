package converter

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var miteminImageIDPattern = regexp.MustCompile(`/icode/([0-9]+)/?`)

// ImageAsset describes a local illustration file needed by the EPUB.
type ImageAsset struct {
	SourcePath string
	Href       string
	MediaType  string
}

// ImageRegistry resolves illustration references and records EPUB assets.
type ImageRegistry struct {
	novelDir string
	assets   map[string]ImageAsset
}

// NewImageRegistry creates an image resolver rooted at a narou.rb novel directory.
func NewImageRegistry(novelDir string) *ImageRegistry {
	return &ImageRegistry{
		novelDir: novelDir,
		assets:   make(map[string]ImageAsset),
	}
}

// Resolve maps an HTML image src to a relative XHTML path and records the asset.
func (r *ImageRegistry) Resolve(src string) (string, error) {
	sourcePath, err := r.resolveSourcePath(src)
	if err != nil {
		return "", err
	}

	mediaType, err := imageMediaType(sourcePath)
	if err != nil {
		return "", err
	}

	href := filepath.ToSlash(filepath.Join("images", filepath.Base(sourcePath)))
	r.assets[href] = ImageAsset{
		SourcePath: sourcePath,
		Href:       href,
		MediaType:  mediaType,
	}

	return "../" + href, nil
}

// Assets returns the resolved images in stable manifest order.
func (r *ImageRegistry) Assets() []ImageAsset {
	assets := make([]ImageAsset, 0, len(r.assets))
	for _, asset := range r.assets {
		assets = append(assets, asset)
	}
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].Href < assets[j].Href
	})

	return assets
}

func (r *ImageRegistry) resolveSourcePath(src string) (string, error) {
	if parsed, err := url.Parse(src); err == nil && parsed.IsAbs() {
		if strings.HasSuffix(parsed.Host, ".mitemin.net") {
			return r.resolveMiteminImage(parsed)
		}
		return "", fmt.Errorf("unsupported image URL: %s", src)
	}

	path := filepath.FromSlash(src)
	if !filepath.IsAbs(path) {
		path = filepath.Join(r.novelDir, path)
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("resolve local image %q: %w", src, err)
	}

	return path, nil
}

func (r *ImageRegistry) resolveMiteminImage(parsed *url.URL) (string, error) {
	matches := miteminImageIDPattern.FindStringSubmatch(parsed.Path)
	if len(matches) != 2 {
		return "", fmt.Errorf("unsupported mitemin URL: %s", parsed.String())
	}

	basename := "i" + matches[1]
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp"} {
		path := filepath.Join(r.novelDir, "挿絵", basename+ext)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("image %s not found in 挿絵", basename)
}

func imageMediaType(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".png":
		return "image/png", nil
	case ".gif":
		return "image/gif", nil
	case ".webp":
		return "image/webp", nil
	default:
		return "", fmt.Errorf("unsupported image type: %s", path)
	}
}
