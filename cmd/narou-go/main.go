package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuanying/narou-go/internal/converter"
	"github.com/yuanying/narou-go/internal/downloader"
	"github.com/yuanying/narou-go/internal/downloader/kakuyomu"
	"github.com/yuanying/narou-go/internal/downloader/syosetu"
	"github.com/yuanying/narou-go/internal/epub"
	"github.com/yuanying/narou-go/internal/library"
	"github.com/yuanying/narou-go/internal/model"
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
		err = runDownload(args[1:], stdout, stderr)
	case "update":
		err = runUpdate(args[1:], stdout, stderr)
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
	libraryPath := flags.String("library", "", "narou.rb library path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	root, err := library.ResolveRoot(*libraryPath)
	if err != nil {
		return err
	}

	db, err := library.LoadDatabase(root)
	if err != nil {
		return err
	}

	for _, entry := range library.Entries(db) {
		_, _ = fmt.Fprintf(stdout, "%d\t%s\t%s\t%s\t%s\n", entry.ID, entryNcode(entry), entry.SiteName, entry.Title, entry.Author)
	}

	return nil
}

func runConvert(args []string, stdout io.Writer) error {
	options, err := parseConvertOptions(args)
	if err != nil {
		return err
	}

	root, err := library.ResolveRoot(options.libraryPath)
	if err != nil {
		return err
	}
	db, err := library.LoadDatabase(root)
	if err != nil {
		return err
	}
	entry, err := library.FindByNcode(db, options.ncode)
	if err != nil {
		return err
	}

	return buildFromLibrary(root, *entry, options.outputPath, options.kindle, stdout)
}

func runDownload(args []string, stdout, stderr io.Writer) error {
	options, err := parseWebOptions("download", args)
	if err != nil {
		return err
	}
	root, err := library.ResolveRoot(options.libraryPath)
	if err != nil {
		return err
	}
	d, err := selectDownloader(options.target)
	if err != nil {
		return err
	}
	db, err := library.LoadDatabase(root)
	if err != nil {
		return err
	}
	if entry, err := library.FindByTarget(db, options.target); err == nil {
		existing, err := library.LoadDownloadedNovel(root, *entry)
		if err != nil {
			return err
		}
		novel, err := d.Update(context.Background(), existing, downloader.WithLog(stderr))
		if err != nil {
			return err
		}

		return saveDownloadedNovel(options, novel, entry, stdout)
	}
	novel, err := d.Download(context.Background(), options.target, downloader.WithLog(stderr))
	if err != nil {
		return err
	}

	return saveDownloadedNovel(options, novel, nil, stdout)
}

func runUpdate(args []string, stdout, stderr io.Writer) error {
	options, err := parseWebOptions("update", args)
	if err != nil {
		return err
	}
	root, err := library.ResolveRoot(options.libraryPath)
	if err != nil {
		return err
	}
	db, err := library.LoadDatabase(root)
	if err != nil {
		return err
	}
	entry, err := library.FindByTarget(db, options.target)
	if err != nil {
		return err
	}
	existing, err := library.LoadDownloadedNovel(root, *entry)
	if err != nil {
		return err
	}
	d, err := selectDownloader(existing.SourceURL)
	if err != nil {
		return err
	}
	novel, err := d.Update(context.Background(), existing, downloader.WithLog(stderr))
	if err != nil {
		return err
	}

	return saveDownloadedNovel(options, novel, entry, stdout)
}

func saveDownloadedNovel(options webOptions, novel *model.Novel, existing *library.NovelEntry, stdout io.Writer) error {
	root, err := library.ResolveRoot(options.libraryPath)
	if err != nil {
		return err
	}
	now := time.Now()
	var entry *library.NovelEntry
	if existing == nil {
		entry, err = library.SaveDownloadedNovel(root, novel, now)
	} else {
		entry, err = library.UpdateDownloadedNovel(root, *existing, novel, now)
	}
	if err != nil {
		return err
	}

	httpClient := downloader.NewHTTPClient()
	imageDir := filepath.Join(library.NovelDir(root, *entry), "挿絵")
	novel.Images = downloader.DownloadImages(context.Background(), httpClient, novel.Images, imageDir)
	if len(novel.Images) > 0 {
		entry, err = library.UpdateDownloadedNovel(root, *entry, novel, now)
		if err != nil {
			return err
		}
	}
	novelDir := library.NovelDir(root, *entry)
	_, _ = fmt.Fprintf(stdout, "saved %s\n", novelDir)

	if options.epub {
		return buildFromLibrary(root, *entry, "", options.kindle, stdout)
	}

	return nil
}

type webOptions struct {
	target      string
	libraryPath string
	epub        bool
	kindle      bool
}

func parseWebOptions(command string, args []string) (webOptions, error) {
	options := webOptions{}
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
		case arg == "--epub":
			options.epub = true
		case arg == "--kindle":
			options.kindle = true
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
	kindle      bool
}

func parseConvertOptions(args []string) (convertOptions, error) {
	options := convertOptions{}
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
		case arg == "--kindle":
			options.kindle = true
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

func buildFromLibrary(root string, entry library.NovelEntry, outputName string, kindle bool, stdout io.Writer) error {
	novelDir := library.NovelDir(root, entry)
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

	outputPath := ebookOutputPath(novelDir, entry, outputName, ".epub")
	if err := buildEPUB(outputPath, epub.Book{
		Title:    toc.Title,
		Author:   toc.Author,
		Language: "ja",
		Sections: sections,
		Images:   epubImages(imageRegistry.Assets()),
	}); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "wrote %s\n", outputPath)
	if kindle {
		kindlePath, err := createKindle(outputPath)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "wrote %s\n", kindlePath)
	}

	return nil
}

func ebookOutputPath(novelDir string, entry library.NovelEntry, outputName string, ext string) string {
	if outputName != "" {
		name := filepath.Base(outputName)
		if filepath.Ext(name) == "" {
			name += ext
		}
		return filepath.Join(novelDir, name)
	}

	return filepath.Join(novelDir, library.EBookFileName(entry, ext))
}

func buildEPUB(outputPath string, book epub.Book) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create epub: %w", err)
	}
	if err := epub.Build(file, book); err != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return fmt.Errorf("build epub: %w; close epub: %w", err, closeErr)
		}
		return fmt.Errorf("build epub: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close epub: %w", err)
	}

	return nil
}

func createKindle(epubPath string) (string, error) {
	outputPath := kindleOutputPath(epubPath)
	cmd := exec.Command("aphrael", epubPath, outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return "", fmt.Errorf("run aphrael: %w", err)
		}
		return "", fmt.Errorf("run aphrael: %w: %s", err, message)
	}

	return outputPath, nil
}

func kindleOutputPath(epubPath string) string {
	ext := filepath.Ext(epubPath)
	if ext == "" {
		return epubPath + ".mobi"
	}

	return strings.TrimSuffix(epubPath, ext) + ".mobi"
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
	_, _ = fmt.Fprintln(stderr, "       narou-go convert ncode [--library path] [--output path] [--kindle]")
	_, _ = fmt.Fprintln(stderr, "       narou-go download [--library path] [--epub] [--kindle] target")
	_, _ = fmt.Fprintln(stderr, "       narou-go update [--library path] [--epub] [--kindle] id")
}
