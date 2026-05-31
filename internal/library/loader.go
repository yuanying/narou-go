package library

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yuanying/narou-go/internal/model"
)

// LoadDownloadedNovel restores a downloaded narou.rb library novel.
func LoadDownloadedNovel(root string, entry NovelEntry) (*model.Novel, error) {
	novelDir := NovelDir(root, entry)
	toc, err := LoadTOC(novelDir)
	if err != nil {
		return nil, err
	}

	novel := &model.Novel{
		ID:        modelIDFromEntry(entry),
		Site:      entry.SiteName,
		SourceURL: firstNonEmpty(toc.TocURL, entry.TocURL),
		Title:     firstNonEmpty(toc.Title, entry.Title),
		Author:    firstNonEmpty(toc.Author, entry.Author),
		Story:     toc.Story,
		NovelType: entry.NovelType,
		End:       entry.End,
		Episodes:  make([]model.Episode, 0, len(toc.Subtitles)),
	}
	for _, subtitle := range toc.Subtitles {
		episode := model.Episode{
			ID:           subtitle.Index,
			Title:        subtitle.Subtitle,
			Chapter:      subtitle.Chapter,
			Subchapter:   subtitle.Subchapter,
			URL:          absoluteURL(novel.SourceURL, subtitle.Href),
			PublishedAt:  subtitle.Subdate,
			UpdatedAt:    subtitle.Subupdate,
			DownloadedAt: subtitle.DownloadTime,
		}
		section, err := LoadSection(novelDir, subtitle)
		if err == nil {
			episode.Preface = section.Element.Introduction
			episode.Body = section.Element.Body
			episode.Afterword = section.Element.Postscript
			episode.BodyHash = hashEpisodeBody(episode.Preface, episode.Body, episode.Afterword)
			if episode.DownloadedAt == "" {
				episode.DownloadedAt = section.DownloadTime
			}
		}
		novel.Episodes = append(novel.Episodes, episode)
	}

	return novel, nil
}

func modelIDFromEntry(entry NovelEntry) string {
	id := entryNcodeForLoader(entry)
	if entry.SiteName == "カクヨム" && !strings.HasPrefix(id, "kakuyomu-") {
		return "kakuyomu-" + id
	}

	return strings.ToLower(id)
}

func entryNcodeForLoader(entry NovelEntry) string {
	if entry.TocURL != "" {
		parts := strings.Split(strings.Trim(strings.TrimPrefix(entry.TocURL, "https://ncode.syosetu.com/"), "/"), "/")
		if len(parts) > 0 && strings.HasPrefix(strings.ToLower(parts[0]), "n") {
			return strings.ToLower(parts[0])
		}
		if strings.Contains(entry.TocURL, "kakuyomu.jp/works/") {
			return strings.TrimPrefix(filepath.Base(strings.TrimSuffix(entry.TocURL, "/")), "works/")
		}
	}
	return strings.Fields(entry.FileTitle)[0]
}

func absoluteURL(base, href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.Contains(base, "ncode.syosetu.com") && strings.HasPrefix(href, "/") {
		return "https://ncode.syosetu.com" + href
	}
	if strings.Contains(base, "kakuyomu.jp") && strings.HasPrefix(href, "/") {
		return "https://kakuyomu.jp" + href
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(href, "/")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

func hashEpisodeBody(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(part))
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}
