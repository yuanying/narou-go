package storage

import (
	"path/filepath"
	"testing"

	"github.com/yuanying/narou-go/internal/model"
)

func TestSaveAndLoadNovel(t *testing.T) {
	root := t.TempDir()
	novel := &model.Novel{
		ID:     "n9669bk",
		Site:   "syosetu",
		Title:  "タイトル",
		Author: "作者",
		Episodes: []model.Episode{
			{ID: "1", Title: "第一話", Body: "<p>本文</p>"},
		},
	}

	if err := SaveNovel(root, novel); err != nil {
		t.Fatalf("SaveNovel() error = %v", err)
	}

	got, err := LoadNovel(root, "n9669bk")
	if err != nil {
		t.Fatalf("LoadNovel() error = %v", err)
	}

	if got.Title != novel.Title || len(got.Episodes) != 1 || got.Episodes[0].Body != "<p>本文</p>" {
		t.Fatalf("LoadNovel() = %#v", got)
	}
	if gotDir := NovelDir(root, "n9669bk"); gotDir != filepath.Join(root, "n9669bk") {
		t.Fatalf("NovelDir() = %q", gotDir)
	}
}
