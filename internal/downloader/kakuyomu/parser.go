package kakuyomu

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/yuanying/narou-go/internal/downloader"
	"github.com/yuanying/narou-go/internal/model"
	xhtml "golang.org/x/net/html"
)

const (
	SiteName = "カクヨム"
	baseURL  = "https://kakuyomu.jp"
)

// ParseWork parses Kakuyomu __NEXT_DATA__ from a work page.
func ParseWork(source string) (*model.Novel, error) {
	doc, err := downloader.ParseHTML(source)
	if err != nil {
		return nil, err
	}
	script := downloader.FindFirst(doc, func(n *xhtml.Node) bool {
		return n.Type == xhtml.ElementNode && n.Data == "script" && downloader.Attr(n, "id") == "__NEXT_DATA__"
	})
	if script == nil || script.FirstChild == nil {
		return nil, fmt.Errorf("__NEXT_DATA__ not found")
	}

	var next nextData
	if err := json.Unmarshal([]byte(script.FirstChild.Data), &next); err != nil {
		return nil, fmt.Errorf("parse __NEXT_DATA__: %w", err)
	}

	workID := next.Query.WorkID
	state := next.Props.PageProps.State
	work := state["Work:"+workID]
	author := state[work.Ref("author")].String("activityName")
	if alternate := work.String("alternateAuthorName"); alternate != "" {
		author = alternate + "／" + author
	}

	novel := &model.Novel{
		ID:        InternalID(workID),
		Site:      SiteName,
		SourceURL: WorkURL(workID),
		Title:     work.String("title"),
		Author:    author,
		Story:     work.String("introduction"),
		NovelType: 1,
		End:       work.String("serialStatus") == "COMPLETED",
	}

	currentChapter := ""
	for _, tocRef := range work.RefListFallback("tableOfContentsV2", "tableOfContents") {
		toc := state[tocRef]
		if chapterRef := toc.Ref("chapter"); chapterRef != "" {
			chapter := state[chapterRef]
			currentChapter = chapter.String("title")
			novel.Chapters = append(novel.Chapters, model.Chapter{Title: currentChapter, Level: chapter.Int("level")})
		}
		for _, episodeRef := range toc.RefList("episodeUnions") {
			episode := state[episodeRef]
			if episode.String("__typename") != "Episode" {
				continue
			}
			id := episode.String("id")
			novel.Episodes = append(novel.Episodes, model.Episode{
				ID:          id,
				Title:       episode.String("title"),
				Chapter:     currentChapter,
				URL:         WorkURL(workID) + "/episodes/" + id,
				PublishedAt: episode.String("publishedAt"),
				UpdatedAt:   episode.String("publishedAt"),
			})
		}
	}
	if len(novel.Episodes) == 0 {
		return nil, fmt.Errorf("no kakuyomu episodes found")
	}

	return novel, nil
}

// ParseEpisode extracts the Kakuyomu episode body and unique image URLs.
func ParseEpisode(source string, episodeURL string) (string, []string, error) {
	doc, err := downloader.ParseHTML(source)
	if err != nil {
		return "", nil, err
	}
	bodyNode := downloader.FindFirst(doc, func(n *xhtml.Node) bool {
		return n.Type == xhtml.ElementNode &&
			n.Data == "div" &&
			downloader.HasClass(n, "widget-episodeBody") &&
			downloader.HasClass(n, "js-episode-body")
	})
	if bodyNode == nil {
		return "", nil, fmt.Errorf("episode body not found")
	}

	return downloader.RenderChildren(bodyNode), extractImageURLs(bodyNode, episodeURL), nil
}

func extractImageURLs(node *xhtml.Node, base string) []string {
	seen := map[string]struct{}{}
	var images []string
	for _, img := range downloader.FindAll(node, downloader.IsElement("img")) {
		src := downloader.Attr(img, "src")
		if src == "" {
			continue
		}
		resolved := resolveURL(base, src)
		if _, ok := seen[resolved]; ok {
			continue
		}
		seen[resolved] = struct{}{}
		images = append(images, resolved)
	}

	return images
}

func resolveURL(base string, ref string) string {
	if strings.HasPrefix(ref, "//") {
		return "https:" + ref
	}
	parsed, err := url.Parse(ref)
	if err != nil || parsed.IsAbs() {
		return ref
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return ref
	}

	return baseURL.ResolveReference(parsed).String()
}

func hashBody(body string) string {
	hash := sha256.Sum256([]byte(body))
	return fmt.Sprintf("%x", hash[:])
}

func imageURLsToModel(urls []string) []model.Image {
	images := make([]model.Image, 0, len(urls))
	for _, url := range urls {
		images = append(images, model.Image{URL: url})
	}

	return images
}

func nowRFC3339() string {
	return time.Now().Format(time.RFC3339)
}

type nextData struct {
	Query struct {
		WorkID string `json:"workId"`
	} `json:"query"`
	Props struct {
		PageProps struct {
			State map[string]stateObject `json:"__APOLLO_STATE__"`
		} `json:"pageProps"`
	} `json:"props"`
}

type stateObject map[string]any

func (s stateObject) String(key string) string {
	value, ok := s[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		return fmt.Sprint(v)
	}
}

func (s stateObject) Int(key string) int {
	value, ok := s[key].(float64)
	if !ok {
		return 0
	}

	return int(value)
}

func (s stateObject) Ref(key string) string {
	obj, ok := s[key].(map[string]any)
	if !ok {
		return ""
	}
	ref, _ := obj["__ref"].(string)

	return ref
}

func (s stateObject) RefList(key string) []string {
	values, ok := s[key].([]any)
	if !ok {
		return nil
	}
	refs := make([]string, 0, len(values))
	for _, value := range values {
		obj, ok := value.(map[string]any)
		if !ok {
			continue
		}
		ref, ok := obj["__ref"].(string)
		if ok {
			refs = append(refs, ref)
		}
	}

	return refs
}

func (s stateObject) RefListFallback(keys ...string) []string {
	for _, key := range keys {
		refs := s.RefList(key)
		if len(refs) > 0 {
			return refs
		}
	}

	return nil
}
