package kakuyomu

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/yuanying/narou-go/internal/downloader"
	"github.com/yuanying/narou-go/internal/model"
)

var kakuyomuInputPattern = regexp.MustCompile(`^https?://kakuyomu\.jp/works/([0-9]+)(?:/episodes/[0-9]+)?/?$`)

// Downloader fetches novels from kakuyomu.jp.
type Downloader struct {
	client *downloader.HTTPClient
}

// New creates a Kakuyomu downloader.
func New(client *downloader.HTTPClient) *Downloader {
	if client == nil {
		client = downloader.NewHTTPClient()
	}

	return &Downloader{client: client}
}

// Match reports whether input is a Kakuyomu work or episode URL.
func (d *Downloader) Match(input string) bool {
	_, err := d.Normalize(input)
	return err == nil
}

// Normalize returns kakuyomu-{workID}.
func (d *Downloader) Normalize(input string) (string, error) {
	workID := WorkID(input)
	if workID == "" {
		return "", fmt.Errorf("invalid kakuyomu input: %s", input)
	}

	return InternalID(workID), nil
}

// Download downloads a Kakuyomu work.
func (d *Downloader) Download(ctx context.Context, input string) (*model.Novel, error) {
	workID := WorkID(input)
	if workID == "" && strings.HasPrefix(input, "kakuyomu-") {
		workID = strings.TrimPrefix(input, "kakuyomu-")
	}
	if workID == "" {
		return nil, fmt.Errorf("invalid kakuyomu input: %s", input)
	}

	source, err := d.client.GetString(ctx, WorkURL(workID))
	if err != nil {
		return nil, err
	}
	novel, err := ParseWork(source)
	if err != nil {
		return nil, err
	}

	var images []model.Image
	for i := range novel.Episodes {
		source, err := d.client.GetString(ctx, novel.Episodes[i].URL)
		if err != nil {
			return nil, err
		}
		body, imageURLs, err := ParseEpisode(source, novel.Episodes[i].URL)
		if err != nil {
			return nil, err
		}
		novel.Episodes[i].Preface = ""
		novel.Episodes[i].Body = body
		novel.Episodes[i].Afterword = ""
		novel.Episodes[i].BodyHash = hashBody(body)
		novel.Episodes[i].DownloadedAt = nowRFC3339()
		images = append(images, imageURLsToModel(imageURLs)...)
	}
	novel.Images = dedupeImages(images)

	return novel, nil
}

// Update redownloads metadata and changed/new episodes.
func (d *Downloader) Update(ctx context.Context, existing *model.Novel) (*model.Novel, error) {
	latest, err := d.Download(ctx, existing.ID)
	if err != nil {
		return nil, err
	}

	oldByID := make(map[string]model.Episode, len(existing.Episodes))
	for _, episode := range existing.Episodes {
		oldByID[episode.ID] = episode
	}
	for i, episode := range latest.Episodes {
		old, ok := oldByID[episode.ID]
		if ok && episode.UpdatedAt != "" && episode.UpdatedAt == old.UpdatedAt && old.BodyHash != "" {
			latest.Episodes[i].Preface = old.Preface
			latest.Episodes[i].Body = old.Body
			latest.Episodes[i].Afterword = old.Afterword
			latest.Episodes[i].BodyHash = old.BodyHash
			latest.Episodes[i].DownloadedAt = old.DownloadedAt
		}
	}

	return latest, nil
}

// WorkID extracts a Kakuyomu work ID.
func WorkID(input string) string {
	if strings.HasPrefix(input, "kakuyomu-") {
		return strings.TrimPrefix(input, "kakuyomu-")
	}
	matches := kakuyomuInputPattern.FindStringSubmatch(strings.TrimSpace(input))
	if len(matches) != 2 {
		return ""
	}

	return matches[1]
}

// InternalID returns the Go storage ID for a work.
func InternalID(workID string) string {
	return "kakuyomu-" + workID
}

// WorkURL returns the Kakuyomu work URL.
func WorkURL(workID string) string {
	return baseURL + "/works/" + workID
}

func dedupeImages(images []model.Image) []model.Image {
	seen := make(map[string]struct{}, len(images))
	out := make([]model.Image, 0, len(images))
	for _, image := range images {
		if image.URL == "" {
			continue
		}
		if _, ok := seen[image.URL]; ok {
			continue
		}
		seen[image.URL] = struct{}{}
		out = append(out, image)
	}

	return out
}
