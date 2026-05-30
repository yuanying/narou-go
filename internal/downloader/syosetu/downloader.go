package syosetu

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/yuanying/narou-go/internal/downloader"
	"github.com/yuanying/narou-go/internal/model"
)

var syosetuInputPattern = regexp.MustCompile(`(?i)^(?:https?://ncode\.syosetu\.com/)?(n\d+[a-z]+)(?:/\d+)?/?$`)

// Downloader fetches novels from ncode.syosetu.com.
type Downloader struct {
	client *downloader.HTTPClient
}

// New creates a syosetu downloader.
func New(client *downloader.HTTPClient) *Downloader {
	if client == nil {
		client = downloader.NewHTTPClient()
	}

	return &Downloader{client: client}
}

// Match reports whether input is a syosetu URL or N-code.
func (d *Downloader) Match(input string) bool {
	_, err := d.Normalize(input)
	return err == nil
}

// Normalize returns a lowercase N-code.
func (d *Downloader) Normalize(input string) (string, error) {
	matches := syosetuInputPattern.FindStringSubmatch(strings.TrimSpace(input))
	if len(matches) != 2 {
		return "", fmt.Errorf("invalid syosetu input: %s", input)
	}

	return strings.ToLower(matches[1]), nil
}

// Download downloads a syosetu novel.
func (d *Downloader) Download(ctx context.Context, input string) (*model.Novel, error) {
	ncode, err := d.Normalize(input)
	if err != nil {
		return nil, err
	}

	source, err := d.client.GetString(ctx, TOCURL(ncode))
	if err != nil {
		return nil, err
	}

	novel, err := ParseTOC(ncode, source)
	if err != nil {
		novel, err = ParseShortStory(ncode, source)
		if err != nil {
			return nil, err
		}
	}
	if novel.NovelType == NovelTypeSeries {
		if err := d.appendPagedTOC(ctx, novel, ncode, source); err != nil {
			return nil, err
		}
	}

	var images []model.Image
	for i := range novel.Episodes {
		if novel.NovelType == NovelTypeShort && novel.Episodes[i].Body != "" {
			images = append(images, novel.Images...)
			continue
		}
		episodeSource, err := d.client.GetString(ctx, novel.Episodes[i].URL)
		if err != nil {
			return nil, err
		}
		element, imageURLs, err := ParseEpisode(episodeSource, novel.Episodes[i].URL)
		if err != nil {
			return nil, err
		}
		novel.Episodes[i].Preface = element.Preface
		novel.Episodes[i].Body = element.Body
		novel.Episodes[i].Afterword = element.Afterword
		novel.Episodes[i].BodyHash = hashBody(element.Preface, element.Body, element.Afterword)
		novel.Episodes[i].DownloadedAt = time.Now().Format(time.RFC3339)
		images = append(images, imageURLsToModel(imageURLs)...)
	}
	novel.Images = dedupeImages(images)

	return novel, nil
}

func (d *Downloader) appendPagedTOC(ctx context.Context, novel *model.Novel, ncode string, source string) error {
	for {
		nextURL, err := NextTOCURL(ncode, source)
		if err != nil {
			return err
		}
		if nextURL == "" {
			return nil
		}
		source, err = d.client.GetString(ctx, nextURL)
		if err != nil {
			return err
		}
		next, err := ParseTOC(ncode, source)
		if err != nil {
			return err
		}
		novel.Chapters = append(novel.Chapters, next.Chapters...)
		novel.Episodes = append(novel.Episodes, next.Episodes...)
	}
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

// TOCURL returns the syosetu table-of-contents URL.
func TOCURL(ncode string) string {
	return "https://ncode.syosetu.com/" + strings.ToLower(ncode) + "/"
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
