package syosetu

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/yuanying/narou-go/internal/downloader"
	"github.com/yuanying/narou-go/internal/model"
	xhtml "golang.org/x/net/html"
)

const (
	SiteName        = "小説家になろう"
	NovelTypeSeries = 1
	NovelTypeShort  = 2
)

// ParseTOC parses a syosetu table-of-contents page.
func ParseTOC(ncode string, source string) (*model.Novel, error) {
	doc, err := downloader.ParseHTML(source)
	if err != nil {
		return nil, err
	}

	novel := &model.Novel{
		ID:        strings.ToLower(ncode),
		Site:      SiteName,
		SourceURL: TOCURL(ncode),
		Title:     firstTextByClass(doc, "p-novel__title"),
		Author:    parseAuthor(doc),
		Story:     firstTextByID(doc, "novel_ex"),
		NovelType: NovelTypeSeries,
	}

	var chapter string
	for child := doc.FirstChild; child != nil; child = nextNode(child) {
		if child.Type != xhtml.ElementNode {
			continue
		}
		if downloader.HasClass(child, "p-eplist__chapter-title") {
			chapter = downloader.TextContent(child)
			novel.Chapters = append(novel.Chapters, model.Chapter{Title: chapter, Level: 1})
			continue
		}
		if !downloader.HasClass(child, "p-eplist__sublist") {
			continue
		}
		episode, ok := parseTOCEpisode(child, chapter, ncode)
		if ok {
			novel.Episodes = append(novel.Episodes, episode)
		}
	}
	if len(novel.Episodes) == 0 {
		return nil, fmt.Errorf("no syosetu episodes found")
	}

	return novel, nil
}

// NextTOCURL returns the next table-of-contents page URL when pagination exists.
func NextTOCURL(ncode string, source string) (string, error) {
	doc, err := downloader.ParseHTML(source)
	if err != nil {
		return "", err
	}
	link := downloader.FindFirst(doc, func(n *xhtml.Node) bool {
		return n.Type == xhtml.ElementNode &&
			n.Data == "a" &&
			downloader.HasClass(n, "c-pager__item") &&
			downloader.HasClass(n, "c-pager__item--next")
	})
	if link == nil {
		return "", nil
	}
	href := downloader.Attr(link, "href")
	if href == "" {
		return "", nil
	}

	return absoluteSyosetuURL(href, ncode), nil
}

// ParseShortStory parses a short story page as a single episode.
func ParseShortStory(ncode string, source string) (*model.Novel, error) {
	doc, err := downloader.ParseHTML(source)
	if err != nil {
		return nil, err
	}
	title := firstTextByClass(doc, "p-novel__title")
	element, images, err := ParseEpisode(source, TOCURL(ncode))
	if err != nil {
		return nil, err
	}

	return &model.Novel{
		ID:        strings.ToLower(ncode),
		Site:      SiteName,
		SourceURL: TOCURL(ncode),
		Title:     title,
		Author:    parseAuthor(doc),
		Story:     firstTextByID(doc, "novel_ex"),
		NovelType: NovelTypeShort,
		Episodes: []model.Episode{{
			ID:           "1",
			Title:        title,
			URL:          TOCURL(ncode),
			Preface:      element.Preface,
			Body:         element.Body,
			Afterword:    element.Afterword,
			BodyHash:     hashBody(element.Preface, element.Body, element.Afterword),
			DownloadedAt: time.Now().Format(time.RFC3339),
		}},
		Images: imageURLsToModel(images),
	}, nil
}

// EpisodeElement is the split HTML body from one episode page.
type EpisodeElement struct {
	Preface   string
	Body      string
	Afterword string
}

// ParseEpisode extracts preface, body, afterword, and image URLs.
func ParseEpisode(source string, episodeURL string) (EpisodeElement, []string, error) {
	doc, err := downloader.ParseHTML(source)
	if err != nil {
		return EpisodeElement{}, nil, err
	}

	element := EpisodeElement{}
	for _, node := range downloader.FindAll(doc, func(node *xhtml.Node) bool {
		return node.Type == xhtml.ElementNode &&
			node.Data == "div" &&
			downloader.HasClass(node, "js-novel-text") &&
			downloader.HasClass(node, "p-novel__text")
	}) {
		content := downloader.RenderChildren(node)
		switch {
		case downloader.HasClass(node, "p-novel__text--preface"):
			element.Preface = content
		case downloader.HasClass(node, "p-novel__text--afterword"):
			element.Afterword = content
		default:
			element.Body = content
		}
	}
	if element.Body == "" {
		return EpisodeElement{}, nil, fmt.Errorf("body not found")
	}

	images := extractImageURLs(doc, episodeURL)
	return element, images, nil
}

func parseTOCEpisode(node *xhtml.Node, chapter string, ncode string) (model.Episode, bool) {
	link := downloader.FindFirst(node, func(n *xhtml.Node) bool {
		return n.Type == xhtml.ElementNode && n.Data == "a" && downloader.HasClass(n, "p-eplist__subtitle")
	})
	if link == nil {
		return model.Episode{}, false
	}
	href := downloader.Attr(link, "href")
	index := episodeIndexFromHref(href)
	if index == "" {
		return model.Episode{}, false
	}

	subdate, subupdate := parseUpdate(node)
	return model.Episode{
		ID:          index,
		Title:       downloader.TextContent(link),
		Chapter:     chapter,
		URL:         absoluteSyosetuURL(href, ncode),
		PublishedAt: subdate,
		UpdatedAt:   subupdate,
	}, true
}

func parseUpdate(node *xhtml.Node) (string, string) {
	update := downloader.FindFirst(node, func(n *xhtml.Node) bool {
		return n.Type == xhtml.ElementNode && downloader.HasClass(n, "p-eplist__update")
	})
	if update == nil {
		return "", ""
	}

	var subupdate string
	span := downloader.FindFirst(update, func(n *xhtml.Node) bool {
		return n.Type == xhtml.ElementNode && n.Data == "span" && strings.Contains(downloader.Attr(n, "title"), "改稿")
	})
	if span != nil {
		subupdate = strings.TrimSuffix(downloader.Attr(span, "title"), " 改稿")
	}
	text := strings.TrimSpace(downloader.TextContent(update))
	text = strings.TrimSpace(strings.Split(text, "（")[0])

	return text, subupdate
}

func firstTextByClass(doc *xhtml.Node, class string) string {
	node := downloader.FindFirst(doc, func(n *xhtml.Node) bool {
		return n.Type == xhtml.ElementNode && downloader.HasClass(n, class)
	})
	if node == nil {
		return ""
	}

	return downloader.TextContent(node)
}

func firstTextByID(doc *xhtml.Node, id string) string {
	node := downloader.FindFirst(doc, func(n *xhtml.Node) bool {
		return n.Type == xhtml.ElementNode && downloader.Attr(n, "id") == id
	})
	if node == nil {
		return ""
	}

	return downloader.TextContent(node)
}

func parseAuthor(doc *xhtml.Node) string {
	author := firstTextByClass(doc, "p-novel__author")
	author = strings.TrimSpace(strings.TrimPrefix(author, "作者："))
	if author != "" {
		return author
	}

	return firstTextByClass(doc, "p-author__name")
}

func extractImageURLs(doc *xhtml.Node, base string) []string {
	seen := map[string]struct{}{}
	var images []string
	for _, img := range downloader.FindAll(doc, downloader.IsElement("img")) {
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

func episodeIndexFromHref(href string) string {
	parts := strings.Split(strings.Trim(href, "/"), "/")
	if len(parts) == 0 {
		return ""
	}

	return parts[len(parts)-1]
}

func absoluteSyosetuURL(href string, ncode string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "/") {
		return "https://ncode.syosetu.com" + href
	}

	return TOCURL(ncode) + href
}

func hashBody(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(part))
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

func imageURLsToModel(urls []string) []model.Image {
	images := make([]model.Image, 0, len(urls))
	for _, url := range urls {
		images = append(images, model.Image{URL: url})
	}

	return images
}

func nextNode(node *xhtml.Node) *xhtml.Node {
	if node.FirstChild != nil {
		return node.FirstChild
	}
	for node != nil {
		if node.NextSibling != nil {
			return node.NextSibling
		}
		node = node.Parent
	}

	return nil
}
