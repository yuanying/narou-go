package epub

import (
	"fmt"
	"strings"
)

func navXHTML(book Book) string {
	var b strings.Builder
	writeBuilder(&b, xhtmlHeader(book.Title, "style/vertical.css"))
	writeBuilder(&b, `<body><nav epub:type="toc" id="toc"><h1>目次</h1><ol>`)
	for i, section := range book.Sections {
		writeBuilder(&b, fmt.Sprintf(`<li><a href="text/p%03d.xhtml">%s</a></li>`, i+1, escapeXML(section.Title)))
	}
	writeBuilder(&b, `</ol></nav></body></html>`)

	return b.String()
}
