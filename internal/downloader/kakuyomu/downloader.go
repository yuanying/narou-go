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
func (d *Downloader) Download(ctx context.Context, input string, opts ...downloader.Option) (*model.Novel, error) {
	options := downloader.NewOptions(opts...)
	return d.download(ctx, input, nil, options)
}

func (d *Downloader) download(ctx context.Context, input string, existing *model.Novel, options downloader.Options) (*model.Novel, error) {
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

	targets := novel.Episodes
	reasons := make(map[string]string, len(targets))
	if existing != nil {
		targets, reasons = downloader.EpisodesToDownload(novel.Episodes, existing.Episodes)
	}
	downloader.LogDownloadStart(options.Log, novel, len(targets))
	var images []model.Image
	for i := range targets {
		target := targets[i]
		index := downloader.FindEpisodeIndex(novel.Episodes, target.ID)
		if index < 0 {
			continue
		}
		downloader.LogEpisodeProgress(options.Log, novel.Episodes[index], i+1, len(targets), reasons[target.ID], novel.NovelType == 1)
		source, err := d.client.GetString(ctx, novel.Episodes[index].URL)
		if err != nil {
			return nil, err
		}
		body, imageURLs, err := ParseEpisode(source, novel.Episodes[index].URL)
		if err != nil {
			return nil, err
		}
		novel.Episodes[index].Preface = ""
		novel.Episodes[index].Body = body
		novel.Episodes[index].Afterword = ""
		novel.Episodes[index].BodyHash = hashBody(body)
		novel.Episodes[index].DownloadedAt = nowRFC3339()
		images = append(images, imageURLsToModel(imageURLs)...)
	}
	if existing != nil {
		downloader.MergeUnchangedEpisodes(novel.Episodes, existing.Episodes)
	}
	novel.Images = dedupeImages(images)

	return novel, nil
}

// Update redownloads metadata and changed/new episodes.
func (d *Downloader) Update(ctx context.Context, existing *model.Novel, opts ...downloader.Option) (*model.Novel, error) {
	options := downloader.NewOptions(opts...)
	return d.download(ctx, existing.ID, existing, options)
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
