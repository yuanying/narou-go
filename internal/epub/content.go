package epub

func contentXHTML(section Section) string {
	return xhtmlHeader(section.Title, "../style/vertical.css") +
		`<body><h1>` + escapeXML(section.Title) + `</h1>` +
		section.Content +
		`</body></html>`
}

func xhtmlHeader(title, stylesheet string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" xml:lang="ja">` +
		`<head><meta charset="UTF-8" /><title>` + escapeXML(title) + `</title>` +
		`<link rel="stylesheet" type="text/css" href="` + escapeXML(stylesheet) + `" />` +
		`</head>`
}
