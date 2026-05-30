package model

import (
	"path/filepath"

	"github.com/yuanying/narou-go/internal/converter"
	"github.com/yuanying/narou-go/internal/epub"
)

// ToEPUBBook converts a downloaded novel to the existing EPUB builder input.
func ToEPUBBook(novel *Novel) epub.Book {
	sections := make([]epub.Section, 0, len(novel.Episodes))
	imageRefs := epubImageRefs(novel.Images)
	for _, episode := range novel.Episodes {
		content := episode.Preface + episode.Body + episode.Afterword
		converted, err := converter.ConvertFragment(content, converter.Options{
			ImageResolver: func(src string) (string, error) {
				if ref, ok := imageRefs[src]; ok {
					return ref, nil
				}
				return src, nil
			},
		})
		if err == nil {
			content = converted
		}
		sections = append(sections, epub.Section{
			Title:   episode.Title,
			Content: content,
		})
	}

	images := make([]epub.Image, 0, len(novel.Images))
	for _, image := range novel.Images {
		if image.Failed || image.LocalPath == "" {
			continue
		}
		images = append(images, epub.Image{
			Href:       "images/" + fileBase(image.LocalPath),
			SourcePath: image.LocalPath,
			MediaType:  image.MediaType,
		})
	}

	return epub.Book{
		Title:    novel.Title,
		Author:   novel.Author,
		Language: "ja",
		Sections: sections,
		Images:   images,
	}
}

func epubImageRefs(images []Image) map[string]string {
	refs := make(map[string]string, len(images))
	for _, image := range images {
		if image.Failed || image.LocalPath == "" {
			continue
		}
		refs[image.URL] = "../images/" + filepath.Base(image.LocalPath)
	}

	return refs
}

func fileBase(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}

	return path
}
