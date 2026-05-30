package model

import (
	"strings"
	"testing"
)

func TestToEPUBBookConvertsHTMLAndImageRefs(t *testing.T) {
	book := ToEPUBBook(&Novel{
		Title:  "タイトル",
		Author: "作者",
		Episodes: []Episode{
			{
				Title: "第一話",
				Body:  `<p id="L1"><b>本文</b><img src="https://example.com/image.jpg" /></p>`,
			},
		},
		Images: []Image{
			{URL: "https://example.com/image.jpg", LocalPath: "/tmp/image.jpg", MediaType: "image/jpeg"},
		},
	})

	if len(book.Sections) != 1 {
		t.Fatalf("len(Sections) = %d, want 1", len(book.Sections))
	}
	content := book.Sections[0].Content
	for _, want := range []string{`<strong>本文</strong>`, `src="../images/image.jpg"`} {
		if !strings.Contains(content, want) {
			t.Fatalf("content missing %q: %s", want, content)
		}
	}
	if len(book.Images) != 1 || book.Images[0].Href != "images/image.jpg" {
		t.Fatalf("Images = %#v", book.Images)
	}
}
