package converter

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func applyTypography(node *html.Node) {
	parent := node.Parent
	if parent == nil {
		return
	}

	replacements := typographyNodes(node.Data)
	if len(replacements) == 1 && replacements[0].Type == html.TextNode && replacements[0].Data == node.Data {
		return
	}

	for _, replacement := range replacements {
		parent.InsertBefore(replacement, node)
	}
	parent.RemoveChild(node)
}

func typographyNodes(text string) []*html.Node {
	nodes := make([]*html.Node, 0, 1)
	var plain strings.Builder

	flushPlain := func() {
		if plain.Len() == 0 {
			return
		}
		nodes = append(nodes, &html.Node{Type: html.TextNode, Data: plain.String()})
		plain.Reset()
	}

	for i := 0; i < len(text); {
		if replacement, width, ok := tcyPunctuation(text[i:]); ok {
			flushPlain()
			nodes = append(nodes, tcyNode(replacement))
			i += width
			continue
		}

		if i+1 < len(text) && isASCIIDigit(text[i]) && isASCIIDigit(text[i+1]) {
			flushPlain()
			nodes = append(nodes, tcyNode(text[i:i+2]))
			i += 2
			continue
		}

		r, width := utf8.DecodeRuneInString(text[i:])
		plain.WriteRune(r)
		i += width
	}
	flushPlain()

	if len(nodes) == 0 {
		return []*html.Node{{Type: html.TextNode, Data: ""}}
	}

	return nodes
}

func tcyPunctuation(text string) (string, int, bool) {
	switch {
	case strings.HasPrefix(text, "！！？"):
		return "!!?", len("！！？"), true
	case strings.HasPrefix(text, "！！"):
		return "!!", len("！！"), true
	case strings.HasPrefix(text, "！？"):
		return "!?", len("！？"), true
	default:
		return "", 0, false
	}
}

func tcyNode(text string) *html.Node {
	span := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Span,
		Data:     "span",
		Attr:     []html.Attribute{{Key: "class", Val: "tcy"}},
	}
	span.AppendChild(&html.Node{Type: html.TextNode, Data: text})
	return span
}

func isASCIIDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func startsWithHalfIndentMarker(node *html.Node) bool {
	text := firstText(node)
	text = strings.TrimLeftFunc(text, unicode.IsSpace)
	if text == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(text)

	switch r {
	case '「', '『', '(', '（', '【':
		return true
	default:
		return false
	}
}

func firstText(node *html.Node) string {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case html.TextNode:
			if child.Data != "" {
				return child.Data
			}
		case html.ElementNode:
			if text := firstText(child); text != "" {
				return text
			}
		}
	}

	return ""
}
