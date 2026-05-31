package downloader

import (
	"context"
	"io"

	"github.com/yuanying/narou-go/internal/model"
)

// Options configures download behavior.
type Options struct {
	Log io.Writer
}

// Option applies a downloader option.
type Option func(*Options)

// WithLog writes progress messages to writer.
func WithLog(writer io.Writer) Option {
	return func(options *Options) {
		options.Log = writer
	}
}

// NewOptions applies opts and returns an options value.
func NewOptions(opts ...Option) Options {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}

	return options
}

// Downloader fetches one web novel site into the common model.
type Downloader interface {
	Match(input string) bool
	Normalize(input string) (string, error)
	Download(ctx context.Context, input string, opts ...Option) (*model.Novel, error)
	Update(ctx context.Context, existing *model.Novel, opts ...Option) (*model.Novel, error)
}
