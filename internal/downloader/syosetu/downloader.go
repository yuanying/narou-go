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
func (d *Downloader) Download(ctx context.Context, input string, opts ...downloader.Option) (*model.Novel, error) {
	options := downloader.NewOptions(opts...)
	return d.download(ctx, input, nil, options)
}

func (d *Downloader) download(ctx context.Context, input string, existing *model.Novel, options downloader.Options) (*model.Novel, error) {
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
		if novel.NovelType == NovelTypeShort && novel.Episodes[index].Body != "" {
			images = append(images, novel.Images...)
			continue
		}
		downloader.LogEpisodeProgress(options.Log, novel.Episodes[index], i+1, len(targets), reasons[target.ID], novel.NovelType == NovelTypeSeries)
		episodeSource, err := d.client.GetString(ctx, novel.Episodes[index].URL)
		if err != nil {
			return nil, err
		}
		element, imageURLs, err := ParseEpisode(episodeSource, novel.Episodes[index].URL)
		if err != nil {
			return nil, err
		}
		novel.Episodes[index].Preface = element.Preface
		novel.Episodes[index].Body = element.Body
		novel.Episodes[index].Afterword = element.Afterword
		novel.Episodes[index].BodyHash = hashBody(element.Preface, element.Body, element.Afterword)
		novel.Episodes[index].DownloadedAt = time.Now().Format(time.RFC3339)
		images = append(images, imageURLsToModel(imageURLs)...)
	}
	if existing != nil {
		downloader.MergeUnchangedEpisodes(novel.Episodes, existing.Episodes)
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
func (d *Downloader) Update(ctx context.Context, existing *model.Novel, opts ...downloader.Option) (*model.Novel, error) {
	options := downloader.NewOptions(opts...)
	return d.download(ctx, existing.ID, existing, options)
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
