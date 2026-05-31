package syosetu

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yuanying/narou-go/internal/downloader"
	"github.com/yuanying/narou-go/internal/model"
)

func TestUpdateDownloadsOnlyChangedSyosetuEpisodes(t *testing.T) {
	var episode1Hits int
	var episode2Hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/n9669bk/":
			_, _ = w.Write([]byte(`<!doctype html><html><body>
<h1 class="p-novel__title">連載タイトル</h1>
<div class="p-novel__author">作者：<a>作者名</a></div>
<div id="novel_ex">あらすじ</div>
<div class="p-eplist__chapter-title">第一章</div>
<div class="p-eplist__sublist"><a href="/n9669bk/1/" class="p-eplist__subtitle">第一話</a><div class="p-eplist__update">2020/01/01 00:00<span title="2020/01/02 12:00 改稿">改</span></div></div>
<div class="p-eplist__sublist"><a href="/n9669bk/2/" class="p-eplist__subtitle">第二話</a><div class="p-eplist__update">2020/01/03 00:00<span title="2020/01/04 12:00 改稿">改</span></div></div>
</body></html>`))
		case "/n9669bk/1/":
			episode1Hits++
			_, _ = w.Write([]byte(readFixture(t, "episode.html")))
		case "/n9669bk/2/":
			episode2Hits++
			_, _ = w.Write([]byte(readFixture(t, "episode.html")))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := downloader.NewHTTPClient()
	client.Wait = 0
	client.Client = server.Client()
	client.Client.Transport = rewriteHostTransport{base: server.URL, next: server.Client().Transport}
	d := New(client)
	existing := &model.Novel{
		ID:        "n9669bk",
		SourceURL: "https://ncode.syosetu.com/n9669bk/",
		Title:     "連載タイトル",
		Author:    "作者名",
		NovelType: NovelTypeSeries,
		Episodes: []model.Episode{
			{ID: "1", Title: "第一話", Chapter: "第一章", URL: "https://ncode.syosetu.com/n9669bk/1/", PublishedAt: "2020/01/01 00:00", UpdatedAt: "2020/01/02 12:00", Body: "old", BodyHash: "hash", DownloadedAt: time.RFC3339},
			{ID: "2", Title: "第二話", Chapter: "第一章", URL: "https://ncode.syosetu.com/n9669bk/2/", PublishedAt: "2020/01/03 00:00", UpdatedAt: "2020/01/03 00:00", Body: "old2", BodyHash: "hash2", DownloadedAt: time.RFC3339},
		},
	}
	var log bytes.Buffer

	latest, err := d.Update(context.Background(), existing, downloader.WithLog(&log))
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if episode1Hits != 0 {
		t.Fatalf("episode 1 was downloaded %d times, want 0", episode1Hits)
	}
	if episode2Hits != 1 {
		t.Fatalf("episode 2 was downloaded %d times, want 1", episode2Hits)
	}
	if latest.Episodes[0].Body != "old" {
		t.Fatalf("unchanged body = %q, want old", latest.Episodes[0].Body)
	}
	if !strings.Contains(log.String(), "更新対象: 1/2") || !strings.Contains(log.String(), "(更新あり)") {
		t.Fatalf("log = %q", log.String())
	}
}

type rewriteHostTransport struct {
	base string
	next http.RoundTripper
}

func (t rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rewritten := req.Clone(req.Context())
	baseReq, _ := http.NewRequest(http.MethodGet, t.base, nil)
	rewritten.URL.Scheme = baseReq.URL.Scheme
	rewritten.URL.Host = baseReq.URL.Host
	return t.next.RoundTrip(rewritten)
}
