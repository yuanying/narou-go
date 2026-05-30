package epub

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Book contains the metadata and text sections needed to build an EPUB.
type Book struct {
	Title    string
	Author   string
	Language string
	Sections []Section
	Images   []Image
}

// Section is one XHTML content document in the EPUB spine.
type Section struct {
	Title   string
	Content string
}

// Image is one local image file embedded into the EPUB.
type Image struct {
	Href       string
	SourcePath string
	MediaType  string
}

// Build writes an EPUB3 archive to w.
func Build(w io.Writer, book Book) error {
	if book.Language == "" {
		book.Language = "ja"
	}

	zipWriter := zip.NewWriter(w)
	if err := writeMimetype(zipWriter); err != nil {
		return err
	}
	if err := writeZipFile(zipWriter, "META-INF/container.xml", containerXML); err != nil {
		return err
	}
	if err := writeZipFile(zipWriter, "OEBPS/package.opf", packageOPF(book)); err != nil {
		return err
	}
	if err := writeZipFile(zipWriter, "OEBPS/nav.xhtml", navXHTML(book)); err != nil {
		return err
	}
	if err := writeZipFile(zipWriter, "OEBPS/style/vertical.css", VerticalCSS); err != nil {
		return err
	}
	for _, image := range book.Images {
		if err := writeImage(zipWriter, image); err != nil {
			return err
		}
	}

	for i, section := range book.Sections {
		name := fmt.Sprintf("OEBPS/text/p%03d.xhtml", i+1)
		if err := writeZipFile(zipWriter, name, contentXHTML(section)); err != nil {
			return err
		}
	}

	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("close epub zip: %w", err)
	}

	return nil
}

func writeMimetype(zipWriter *zip.Writer) error {
	header := &zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	}
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create mimetype: %w", err)
	}
	if _, err := writer.Write([]byte("application/epub+zip")); err != nil {
		return fmt.Errorf("write mimetype: %w", err)
	}

	return nil
}

func writeZipFile(zipWriter *zip.Writer, name, content string) error {
	writer, err := zipWriter.Create(name)
	if err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}
	if _, err := writer.Write([]byte(content)); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}

	return nil
}

func writeImage(zipWriter *zip.Writer, image Image) error {
	data, err := os.ReadFile(image.SourcePath)
	if err != nil {
		return fmt.Errorf("read image %s: %w", image.SourcePath, err)
	}

	name := filepath.ToSlash(filepath.Join("OEBPS", image.Href))
	writer, err := zipWriter.Create(name)
	if err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}

	return nil
}

func coverImageIndex(images []Image) int {
	if len(images) == 0 {
		return -1
	}
	for i, image := range images {
		name := strings.ToLower(filepath.Base(image.Href))
		if name == "cover.jpg" || name == "cover.jpeg" || name == "cover.png" {
			return i
		}
	}

	return 0
}
