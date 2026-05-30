package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yuanying/narou-go/internal/converter"
	"github.com/yuanying/narou-go/internal/downloader"
	"github.com/yuanying/narou-go/internal/downloader/kakuyomu"
	"github.com/yuanying/narou-go/internal/downloader/syosetu"
	"github.com/yuanying/narou-go/internal/epub"
	"github.com/yuanying/narou-go/internal/library"
	"github.com/yuanying/narou-go/internal/model"
	"github.com/yuanying/narou-go/internal/storage"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeUsage(stderr)
		return 2
	}

	var err error
	switch args[0] {
	case "list":
		err = runList(args[1:], stdout)
	case "convert":
		err = runConvert(args[1:], stdout)
	case "download":
		err = runDownload(args[1:], stdout)
	case "update":
		err = runUpdate(args[1:], stdout)
	default:
		writeUsage(stderr)
		return 2
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	return 0
}

func runList(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	libraryPath := flags.String("library", "./library", "narou.rb library path")
	if err := flags.Parse(args); err != nil {
		return err
	}

	db, err := library.LoadDatabase(*libraryPath)
	if err != nil {
		return err
	}

	entries := make([]library.NovelEntry, 0, len(db))
	for _, entry := range db {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})

	for _, entry := range entries {
		_, _ = fmt.Fprintf(stdout, "%d\t%s\t%s\t%s\t%s\n", entry.ID, entryNcode(entry), entry.SiteName, entry.Title, entry.Author)
	}

	return nil
}

func runConvert(args []string, stdout io.Writer) error {
	options, err := parseConvertOptions(args)
	if err != nil {
		return err
	}

	db, err := library.LoadDatabase(options.libraryPath)
	if err != nil {
		return err
	}
	entry, err := library.FindByNcode(db, options.ncode)
	if err != nil {
		return err
	}

	novelDir := library.NovelDir(options.libraryPath, *entry)
	toc, err := library.LoadTOC(novelDir)
	if err != nil {
		return err
	}

	imageRegistry := converter.NewImageRegistry(novelDir)
	sections := make([]epub.Section, 0, len(toc.Subtitles))
	for _, subtitle := range toc.Subtitles {
		section, err := library.LoadSection(novelDir, subtitle)
		if err != nil {
			return err
		}
		content, err := convertSectionContent(section, imageRegistry)
		if err != nil {
			return err
		}
		sections = append(sections, epub.Section{
			Title:   section.Subtitle,
			Content: content,
		})
	}

	outputPath := options.outputPath
	if outputPath == "" {
		outputPath = entry.Title + ".epub"
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	if err := epub.Build(file, epub.Book{
		Title:    toc.Title,
		Author:   toc.Author,
		Language: "ja",
		Sections: sections,
		Images:   epubImages(imageRegistry.Assets()),
	}); err != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return fmt.Errorf("build epub: %w; close output: %w", err, closeErr)
		}
		return err
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close output: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "wrote %s\n", outputPath)
	return nil
}

func runDownload(args []string, stdout io.Writer) error {
	options, err := parseWebOptions("download", args)
	if err != nil {
		return err
	}
	d, err := selectDownloader(options.target)
	if err != nil {
		return err
	}
	novel, err := d.Download(context.Background(), options.target)
	if err != nil {
		return err
	}

	return saveDownloadedNovel(options, novel, stdout)
}

func runUpdate(args []string, stdout io.Writer) error {
	options, err := parseWebOptions("update", args)
	if err != nil {
		return err
	}
	existing, err := storage.LoadNovel(options.dataPath, options.target)
	if err != nil {
		return err
	}
	d, err := selectDownloader(existing.SourceURL)
	if err != nil {
		return err
	}
	novel, err := d.Update(context.Background(), existing)
	if err != nil {
		return err
	}

	return saveDownloadedNovel(options, novel, stdout)
}

func saveDownloadedNovel(options webOptions, novel *model.Novel, stdout io.Writer) error {
	httpClient := downloader.NewHTTPClient()
	imageDir := filepath.Join(storage.NovelDir(options.dataPath, novel.ID), "images")
	novel.Images = downloader.DownloadImages(context.Background(), httpClient, novel.Images, imageDir)
	if err := storage.SaveNovel(options.dataPath, novel); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "saved %s\n", storage.NovelDir(options.dataPath, novel.ID))

	if options.epub {
		outputPath := filepath.Join(storage.NovelDir(options.dataPath, novel.ID), novel.ID+".epub")
		file, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("create epub: %w", err)
		}
		if err := epub.Build(file, model.ToEPUBBook(novel)); err != nil {
			closeErr := file.Close()
			if closeErr != nil {
				return fmt.Errorf("build epub: %w; close epub: %w", err, closeErr)
			}
			return err
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("close epub: %w", err)
		}
		_, _ = fmt.Fprintf(stdout, "wrote %s\n", outputPath)
	}

	return nil
}

type webOptions struct {
	target   string
	dataPath string
	epub     bool
}

func parseWebOptions(command string, args []string) (webOptions, error) {
	options := webOptions{dataPath: "./data"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--data":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--data requires a value")
			}
			options.dataPath = args[i+1]
			i++
		case strings.HasPrefix(arg, "--data="):
			options.dataPath = strings.TrimPrefix(arg, "--data=")
		case arg == "--epub":
			options.epub = true
		case strings.HasPrefix(arg, "-"):
			return options, fmt.Errorf("unknown option: %s", arg)
		case options.target == "":
			options.target = arg
		default:
			return options, fmt.Errorf("unexpected argument: %s", arg)
		}
	}
	if options.target == "" {
		return options, fmt.Errorf("%s requires target", command)
	}

	return options, nil
}

func selectDownloader(input string) (downloader.Downloader, error) {
	downloaders := []downloader.Downloader{
		syosetu.New(nil),
		kakuyomu.New(nil),
	}
	for _, d := range downloaders {
		if d.Match(input) {
			return d, nil
		}
	}

	return nil, fmt.Errorf("unsupported download target: %s", input)
}

type convertOptions struct {
	ncode       string
	libraryPath string
	outputPath  string
}

func parseConvertOptions(args []string) (convertOptions, error) {
	options := convertOptions{libraryPath: "./library"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--library":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--library requires a value")
			}
			options.libraryPath = args[i+1]
			i++
		case strings.HasPrefix(arg, "--library="):
			options.libraryPath = strings.TrimPrefix(arg, "--library=")
		case arg == "--output":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--output requires a value")
			}
			options.outputPath = args[i+1]
			i++
		case strings.HasPrefix(arg, "--output="):
			options.outputPath = strings.TrimPrefix(arg, "--output=")
		case strings.HasPrefix(arg, "-"):
			return options, fmt.Errorf("unknown option: %s", arg)
		case options.ncode == "":
			options.ncode = arg
		default:
			return options, fmt.Errorf("unexpected argument: %s", arg)
		}
	}
	if options.ncode == "" {
		return options, fmt.Errorf("convert requires ncode")
	}

	return options, nil
}

func convertSectionContent(section *library.Section, resolver *converter.ImageRegistry) (string, error) {
	fragments := []string{section.Element.Introduction, section.Element.Body, section.Element.Postscript}
	var content strings.Builder
	for _, fragment := range fragments {
		if fragment == "" {
			continue
		}
		converted, err := converter.ConvertFragment(fragment, converter.Options{ImageResolver: resolver.Resolve})
		if err != nil {
			return "", err
		}
		_, _ = content.WriteString(converted)
	}

	return content.String(), nil
}

func epubImages(assets []converter.ImageAsset) []epub.Image {
	images := make([]epub.Image, 0, len(assets))
	for _, asset := range assets {
		images = append(images, epub.Image{
			Href:       asset.Href,
			SourcePath: asset.SourcePath,
			MediaType:  asset.MediaType,
		})
	}

	return images
}

func entryNcode(entry library.NovelEntry) string {
	fields := strings.Fields(entry.FileTitle)
	if len(fields) > 0 {
		return fields[0]
	}

	return filepath.Base(entry.TocURL)
}

func writeUsage(stderr io.Writer) {
	_, _ = fmt.Fprintln(stderr, "usage: narou-go list [--library path]")
	_, _ = fmt.Fprintln(stderr, "       narou-go convert ncode [--library path] [--output path]")
	_, _ = fmt.Fprintln(stderr, "       narou-go download [--data path] [--epub] target")
	_, _ = fmt.Fprintln(stderr, "       narou-go update [--data path] [--epub] id")
}
