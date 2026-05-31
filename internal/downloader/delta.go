package downloader

import (
	"fmt"
	"io"

	"github.com/yuanying/narou-go/internal/model"
)

// EpisodesToDownload returns episodes that must be downloaded and reason labels.
func EpisodesToDownload(latest, existing []model.Episode) ([]model.Episode, map[string]string) {
	oldByID := make(map[string]model.Episode, len(existing))
	for _, episode := range existing {
		oldByID[episode.ID] = episode
	}
	targets := make([]model.Episode, 0)
	reasons := make(map[string]string)
	for _, episode := range latest {
		old, ok := oldByID[episode.ID]
		reason := UpdateReason(episode, old, ok)
		if reason == "" {
			continue
		}
		targets = append(targets, episode)
		reasons[episode.ID] = reason
	}

	return targets, reasons
}

// UpdateReason returns an empty string when the old episode can be reused.
func UpdateReason(latest, old model.Episode, ok bool) string {
	if !ok {
		return "new"
	}
	if old.BodyHash == "" {
		return "missing"
	}
	if old.Title != latest.Title || old.Chapter != latest.Chapter || old.Subchapter != latest.Subchapter {
		return "updated"
	}
	if latest.UpdatedAt != "" || old.UpdatedAt != "" {
		if latest.UpdatedAt != old.UpdatedAt {
			return "updated"
		}
		return ""
	}
	if latest.PublishedAt != "" || old.PublishedAt != "" {
		if latest.PublishedAt != old.PublishedAt {
			return "updated"
		}
		return ""
	}

	return "updated"
}

// MergeUnchangedEpisodes copies old bodies into latest episodes that were not downloaded.
func MergeUnchangedEpisodes(latest, existing []model.Episode) {
	oldByID := make(map[string]model.Episode, len(existing))
	for _, episode := range existing {
		oldByID[episode.ID] = episode
	}
	for i, episode := range latest {
		old, ok := oldByID[episode.ID]
		if !ok || latest[i].BodyHash != "" {
			continue
		}
		latest[i].Preface = old.Preface
		latest[i].Body = old.Body
		latest[i].Afterword = old.Afterword
		latest[i].BodyHash = old.BodyHash
		latest[i].DownloadedAt = old.DownloadedAt
	}
}

// FindEpisodeIndex returns the index for id, or -1.
func FindEpisodeIndex(episodes []model.Episode, id string) int {
	for i, episode := range episodes {
		if episode.ID == id {
			return i
		}
	}

	return -1
}

// LogDownloadStart writes narou.rb-style progress messages.
func LogDownloadStart(writer io.Writer, novel *model.Novel, targetCount int) {
	if writer == nil {
		return
	}
	_, _ = fmt.Fprintf(writer, "ID:%s %s のDL開始\n", novel.ID, novel.Title)
	_, _ = fmt.Fprintf(writer, "更新対象: %d/%d\n", targetCount, len(novel.Episodes))
	if targetCount == 0 {
		_, _ = fmt.Fprintf(writer, "%s に更新はありません\n", novel.Title)
	}
}

// LogEpisodeProgress writes one episode progress line.
func LogEpisodeProgress(writer io.Writer, episode model.Episode, current, total int, reason string, series bool) {
	if writer == nil {
		return
	}
	if episode.Chapter != "" {
		_, _ = fmt.Fprintln(writer, episode.Chapter)
	}
	if series {
		_, _ = fmt.Fprintf(writer, "第%s部分 ", episode.ID)
	} else {
		_, _ = fmt.Fprint(writer, "短編 ")
	}
	_, _ = fmt.Fprintf(writer, "%s (%d/%d)", episode.Title, current, total)
	switch reason {
	case "new":
		_, _ = fmt.Fprint(writer, " (新着)")
	case "updated", "missing":
		_, _ = fmt.Fprint(writer, " (更新あり)")
	}
	_, _ = fmt.Fprintln(writer)
}
