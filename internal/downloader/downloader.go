package downloader

import (
	"context"

	"github.com/yuanying/narou-go/internal/model"
)

// Downloader fetches one web novel site into the common model.
type Downloader interface {
	Match(input string) bool
	Normalize(input string) (string, error)
	Download(ctx context.Context, input string) (*model.Novel, error)
	Update(ctx context.Context, existing *model.Novel) (*model.Novel, error)
}
