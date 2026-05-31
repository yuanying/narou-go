package library

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yuanying/narou-go/internal/model"
	"gopkg.in/yaml.v3"
)

const (
	archiveRootDirName  = "小説データ"
	bodyDirName         = "本文"
	illustrationDirName = "挿絵"
)

var (
	rubyRtPattern  = regexp.MustCompile(`<rt>.*?</rt>`)
	rubyTagPattern = regexp.MustCompile(`</?ruby>|</?rb>`)
)

// SaveDownloadedNovel writes a downloaded novel in narou.rb library format.
func SaveDownloadedNovel(root string, novel *model.Novel, now time.Time) (*NovelEntry, error) {
	db, err := loadDatabaseRaw(root)
	if err != nil {
		return nil, err
	}
	id := nextDatabaseID(db)
	entry := newNovelEntry(id, "", novel)
	if err := writeNovelFiles(root, entry, novel, now); err != nil {
		return nil, err
	}
	db[id] = databaseRecord(entry, novel, now)
	if err := saveDatabaseRaw(root, db); err != nil {
		return nil, err
	}

	return &entry, nil
}

// UpdateDownloadedNovel overwrites an existing narou.rb library novel entry.
func UpdateDownloadedNovel(root string, existing NovelEntry, novel *model.Novel, now time.Time) (*NovelEntry, error) {
	db, err := loadDatabaseRaw(root)
	if err != nil {
		return nil, err
	}
	entry := newNovelEntry(existing.ID, existing.FileTitle, novel)
	if existing.SiteName != "" {
		entry.SiteName = existing.SiteName
	}
	if err := writeNovelFiles(root, entry, novel, now); err != nil {
		return nil, err
	}
	record := db[entry.ID]
	if record == nil {
		record = map[string]any{}
	}
	for key, value := range databaseRecord(entry, novel, now) {
		record[key] = value
	}
	db[entry.ID] = record
	if err := saveDatabaseRaw(root, db); err != nil {
		return nil, err
	}

	return &entry, nil
}

func writeNovelFiles(root string, entry NovelEntry, novel *model.Novel, now time.Time) error {
	novelDir := NovelDir(root, entry)
	if err := os.MkdirAll(filepath.Join(novelDir, bodyDirName), 0o755); err != nil {
		return fmt.Errorf("create novel body dir: %w", err)
	}

	toc := TOC{
		Title:     novel.Title,
		Author:    novel.Author,
		TocURL:    novel.SourceURL,
		Story:     novel.Story,
		Subtitles: make([]Subtitle, 0, len(novel.Episodes)),
	}
	expectedSections := map[string]struct{}{}
	for _, episode := range novel.Episodes {
		subtitle := subtitleFromEpisode(episode, now)
		toc.Subtitles = append(toc.Subtitles, subtitle)
		section := Section{
			Index:        subtitle.Index,
			Href:         subtitle.Href,
			Chapter:      subtitle.Chapter,
			Subchapter:   subtitle.Subchapter,
			Subtitle:     subtitle.Subtitle,
			FileSubtitle: subtitle.FileSubtitle,
			Subdate:      subtitle.Subdate,
			Subupdate:    subtitle.Subupdate,
			DownloadTime: subtitle.DownloadTime,
			Element: Element{
				DataType:     "html",
				Introduction: localizeImageRefs(episode.Preface, novel.Images),
				Body:         localizeImageRefs(episode.Body, novel.Images),
				Postscript:   localizeImageRefs(episode.Afterword, novel.Images),
			},
		}
		name := sectionFileName(subtitle)
		expectedSections[name] = struct{}{}
		if err := writeYAML(filepath.Join(novelDir, bodyDirName, name), section, false); err != nil {
			return err
		}
	}
	if err := removeStaleSections(filepath.Join(novelDir, bodyDirName), expectedSections); err != nil {
		return err
	}
	if err := writeYAML(filepath.Join(novelDir, "toc.yaml"), toc, true); err != nil {
		return err
	}

	return nil
}

func loadDatabaseRaw(root string) (map[int]map[string]any, error) {
	path := filepath.Join(root, localSettingDirName, "database.yaml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[int]map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read database: %w", err)
	}
	db := map[int]map[string]any{}
	if err := yaml.Unmarshal(data, &db); err != nil {
		return nil, fmt.Errorf("parse database: %w", err)
	}

	return db, nil
}

func saveDatabaseRaw(root string, db map[int]map[string]any) error {
	return writeYAML(filepath.Join(root, localSettingDirName, "database.yaml"), db, true)
}

func nextDatabaseID(db map[int]map[string]any) int {
	maxID := -1
	for id := range db {
		if id > maxID {
			maxID = id
		}
	}

	return maxID + 1
}

func newNovelEntry(id int, existingFileTitle string, novel *model.Novel) NovelEntry {
	siteName := siteName(novel)
	fileTitle := existingFileTitle
	if fileTitle == "" {
		fileTitle = makeFileTitle(novel)
	}
	entry := NovelEntry{
		ID:        id,
		Author:    novel.Author,
		Title:     novel.Title,
		FileTitle: fileTitle,
		TocURL:    novel.SourceURL,
		SiteName:  siteName,
		NovelType: novel.NovelType,
		End:       novel.End,
		Tags:      tagsForNovel(novel),
	}
	if entry.NovelType == 0 {
		entry.NovelType = 1
	}

	return entry
}

func databaseRecord(entry NovelEntry, novel *model.Novel, now time.Time) map[string]any {
	firstUp := parseEpisodeTime(firstEpisode(novel).PublishedAt, now)
	lastUp := parseEpisodeTime(lastEpisode(novel).UpdatedAt, parseEpisodeTime(lastEpisode(novel).PublishedAt, now))
	return map[string]any{
		"id":                entry.ID,
		"author":            entry.Author,
		"title":             entry.Title,
		"file_title":        entry.FileTitle,
		"toc_url":           entry.TocURL,
		"sitename":          entry.SiteName,
		"novel_type":        entry.NovelType,
		"end":               entry.End,
		"last_update":       now,
		"new_arrivals_date": now,
		"use_subdirectory":  false,
		"general_firstup":   firstUp,
		"novelupdated_at":   lastUp,
		"general_lastup":    lastUp,
		"length":            0,
		"suspend":           false,
		"general_all_no":    len(novel.Episodes),
		"last_check_date":   now,
		"tags":              entry.Tags,
	}
}

func firstEpisode(novel *model.Novel) model.Episode {
	if len(novel.Episodes) == 0 {
		return model.Episode{}
	}

	return novel.Episodes[0]
}

func lastEpisode(novel *model.Novel) model.Episode {
	if len(novel.Episodes) == 0 {
		return model.Episode{}
	}

	return novel.Episodes[len(novel.Episodes)-1]
}

func subtitleFromEpisode(episode model.Episode, now time.Time) Subtitle {
	fileSubtitle := titleToFilename(episode.Title)
	return Subtitle{
		Index:        episode.ID,
		Href:         hrefFromURL(episode.URL),
		Chapter:      episode.Chapter,
		Subchapter:   episode.Subchapter,
		Subtitle:     strings.ReplaceAll(episode.Title, "\n", ""),
		FileSubtitle: fileSubtitle,
		Subdate:      episode.PublishedAt,
		Subupdate:    episode.UpdatedAt,
		DownloadTime: downloadTime(episode, now),
	}
}

func sectionFileName(subtitle Subtitle) string {
	return subtitle.Index + " " + subtitle.FileSubtitle + ".yaml"
}

func removeStaleSections(bodyDir string, expected map[string]struct{}) error {
	entries, err := os.ReadDir(bodyDir)
	if err != nil {
		return fmt.Errorf("read body dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		if _, ok := expected[entry.Name()]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(bodyDir, entry.Name())); err != nil {
			return fmt.Errorf("remove stale section: %w", err)
		}
	}

	return nil
}

func writeYAML(path string, value any, backup bool) error {
	data, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create yaml dir: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp yaml: %w", err)
	}
	tempPath := temp.Name()
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("write temp yaml: %w", err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close temp yaml: %w", err)
	}
	if backup {
		if err := os.WriteFile(path+".backup", data, 0o644); err != nil {
			_ = os.Remove(tempPath)
			return fmt.Errorf("write yaml backup: %w", err)
		}
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace yaml: %w", err)
	}

	return nil
}

func siteName(novel *model.Novel) string {
	if strings.HasPrefix(novel.ID, "kakuyomu-") || strings.Contains(novel.SourceURL, "kakuyomu.jp") || novel.Site == "kakuyomu" {
		return "カクヨム"
	}

	return "小説家になろう"
}

func makeFileTitle(novel *model.Novel) string {
	id := novel.ID
	id = strings.TrimPrefix(id, "kakuyomu-")
	return truncateFolderTitle(id + " " + replaceFilenameSpecialChars(novel.Title))
}

func titleToFilename(title string) string {
	title = rubyRtPattern.ReplaceAllString(title, "")
	title = rubyTagPattern.ReplaceAllString(title, "")
	return truncatePath(replaceFilenameSpecialChars(title))
}

func replaceFilenameSpecialChars(value string) string {
	replacer := strings.NewReplacer(
		"/", "／",
		":", "：",
		"*", "＊",
		"?", "？",
		"\"", "”",
		"<", "〈",
		">", "〉",
		"[", "［",
		"]", "］",
		"{", "｛",
		"}", "｝",
		"|", "｜",
		".", "．",
		"`", "｀",
		"\\", "￥",
		"\t", "",
		"\n", "",
	)
	value = replacer.Replace(value)
	value = strings.TrimSpace(value)
	if value == "" {
		return "untitled"
	}

	return value
}

func truncateFolderTitle(title string) string {
	runes := []rune(title)
	if len(runes) > 50 {
		runes = runes[:50]
	}

	return strings.TrimSpace(string(runes))
}

func truncatePath(name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	runes := []rune(base)
	if len(runes) > 50 {
		base = string(runes[:50])
	}

	return base + ext
}

func tagsForNovel(novel *model.Novel) []string {
	if novel.End {
		return []string{"end"}
	}

	return []string{}
}

func hrefFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || !parsed.IsAbs() {
		return rawURL
	}
	if parsed.RawQuery != "" {
		return parsed.EscapedPath() + "?" + parsed.RawQuery
	}

	return parsed.EscapedPath()
}

func downloadTime(episode model.Episode, now time.Time) string {
	if episode.DownloadedAt != "" {
		return episode.DownloadedAt
	}

	return now.Format(time.RFC3339)
}

func parseEpisodeTime(value string, fallback time.Time) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006/01/02 15:04",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05.000000000 -07:00",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}

	return fallback
}

func localizeImageRefs(fragment string, images []model.Image) string {
	for _, image := range images {
		if image.URL == "" || image.LocalPath == "" || strings.Contains(image.URL, ".mitemin.net") {
			continue
		}
		fragment = strings.ReplaceAll(fragment, image.URL, filepath.ToSlash(filepath.Join(illustrationDirName, filepath.Base(image.LocalPath))))
	}

	return fragment
}

// FindByTarget finds a database entry by URL, ncode, work ID, or file_title prefix.
func FindByTarget(db map[int]NovelEntry, target string) (*NovelEntry, error) {
	normalized := strings.ToLower(strings.TrimSpace(target))
	if normalized == "" {
		return nil, fmt.Errorf("target is empty")
	}
	if parsed, err := url.Parse(normalized); err == nil && parsed.IsAbs() {
		normalized = strings.TrimSuffix(parsed.String(), "/")
	}
	for _, entry := range db {
		tocURL := strings.TrimSuffix(strings.ToLower(entry.TocURL), "/")
		fileTitle := strings.ToLower(entry.FileTitle)
		if tocURL == normalized ||
			fileTitle == normalized ||
			strings.HasPrefix(fileTitle, normalized+" ") ||
			strings.Contains(tocURL, "/"+normalized) ||
			strings.Contains(fileTitle, normalized) {
			found := entry
			return &found, nil
		}
	}
	if _, err := strconv.Atoi(normalized); err == nil {
		for _, entry := range db {
			if entry.ID == atoi(normalized) {
				found := entry
				return &found, nil
			}
		}
	}

	return nil, fmt.Errorf("target %q not found", target)
}

func atoi(value string) int {
	number, _ := strconv.Atoi(value)
	return number
}

func Entries(db map[int]NovelEntry) []NovelEntry {
	entries := make([]NovelEntry, 0, len(db))
	for _, entry := range db {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})

	return entries
}

// EBookFileName returns the default narou.rb-style ebook file name.
func EBookFileName(entry NovelEntry, ext string) string {
	author := replaceFilenameSpecialChars(entry.Author)
	title := replaceFilenameSpecialChars(entry.Title)
	return truncatePath("[" + author + "] " + title + ext)
}
