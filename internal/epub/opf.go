package epub

import (
	"fmt"
	"strings"
)

func packageOPF(book Book) string {
	var b strings.Builder
	writeBuilder(&b, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	writeBuilder(&b, `<package version="3.0" unique-identifier="book-id" xmlns="http://www.idpf.org/2007/opf">`+"\n")
	writeBuilder(&b, `  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">`+"\n")
	writeBuilder(&b, `    <dc:identifier id="book-id">narou-go</dc:identifier>`+"\n")
	writeBuilder(&b, `    <dc:title>`+escapeXML(book.Title)+`</dc:title>`+"\n")
	writeBuilder(&b, `    <dc:creator>`+escapeXML(book.Author)+`</dc:creator>`+"\n")
	writeBuilder(&b, `    <dc:language>`+escapeXML(book.Language)+`</dc:language>`+"\n")
	writeBuilder(&b, `    <meta name="primary-writing-mode" content="vertical-rl"></meta>`+"\n")
	writeBuilder(&b, `    <meta property="rendition:layout">reflowable</meta>`+"\n")
	writeBuilder(&b, `    <meta property="rendition:orientation">auto</meta>`+"\n")
	writeBuilder(&b, `    <meta property="rendition:spread">auto</meta>`+"\n")
	writeBuilder(&b, "  </metadata>\n")
	writeBuilder(&b, "  <manifest>\n")
	writeBuilder(&b, `    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"></item>`+"\n")
	writeBuilder(&b, `    <item id="style" href="style/vertical.css" media-type="text/css"></item>`+"\n")
	writeBuilder(&b, `    <item id="cover-page" href="cover.xhtml" media-type="application/xhtml+xml"></item>`+"\n")
	for i := range book.Sections {
		writeBuilder(&b, fmt.Sprintf(`    <item id="p%03d" href="text/p%03d.xhtml" media-type="application/xhtml+xml"></item>`+"\n", i+1, i+1))
	}
	coverIndex := coverImageIndex(book.Images)
	for i, image := range book.Images {
		properties := ""
		if i == coverIndex {
			properties = ` properties="cover-image"`
		}
		writeBuilder(&b, fmt.Sprintf(`    <item id="image-%03d" href="%s" media-type="%s"%s></item>`+"\n", i+1, escapeXML(image.Href), escapeXML(image.MediaType), properties))
	}
	writeBuilder(&b, "  </manifest>\n")
	writeBuilder(&b, `  <spine page-progression-direction="rtl">`+"\n")
	writeBuilder(&b, `    <itemref idref="cover-page"></itemref>`+"\n")
	for i := range book.Sections {
		writeBuilder(&b, fmt.Sprintf(`    <itemref idref="p%03d"></itemref>`+"\n", i+1))
	}
	writeBuilder(&b, "  </spine>\n")
	writeBuilder(&b, "</package>")

	return b.String()
}

func writeBuilder(b *strings.Builder, s string) {
	_, _ = b.WriteString(s)
}
