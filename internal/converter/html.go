package converter

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ImageResolver maps a source URL in narou.rb HTML to an EPUB-local path.
type ImageResolver func(src string) (string, error)

// Options configures HTML fragment conversion.
type Options struct {
	ImageResolver ImageResolver
}

// ConvertFragment converts narou.rb HTML fragments into EPUB XHTML fragments.
func ConvertFragment(input string, options Options) (string, error) {
	context := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Body,
		Data:     "body",
	}
	nodes, err := html.ParseFragment(strings.NewReader(input), context)
	if err != nil {
		return "", fmt.Errorf("parse html fragment: %w", err)
	}

	for _, node := range nodes {
		if err := transformNode(node, options); err != nil {
			return "", err
		}
	}

	var out strings.Builder
	for _, node := range nodes {
		renderXHTML(&out, node)
	}

	return out.String(), nil
}

func transformNode(node *html.Node, options Options) error {
	if node.Type == html.TextNode {
		applyTypography(node)
		return nil
	}

	if node.Type == html.ElementNode {
		if err := transformElement(node, options); err != nil {
			return err
		}
	}

	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		if err := transformNode(child, options); err != nil {
			return err
		}
		child = next
	}

	if node.Type == html.ElementNode && node.Data == "p" && startsWithHalfIndentMarker(node) {
		addClass(node, "half-indent")
	}

	return nil
}

func transformElement(node *html.Node, options Options) error {
	switch node.Data {
	case "p":
		removeAttr(node, "id")
	case "b":
		node.Data = "strong"
		node.DataAtom = atom.Strong
	case "i":
		node.Data = "em"
		node.DataAtom = atom.Em
	case "s":
		node.Data = "span"
		node.DataAtom = atom.Span
		setAttr(node, "class", "strikethrough")
	case "img":
		if options.ImageResolver != nil {
			src := attr(node, "src")
			resolved, err := options.ImageResolver(src)
			if err != nil {
				return fmt.Errorf("resolve image %q: %w", src, err)
			}
			setAttr(node, "src", resolved)
		}
	case "a":
		unwrap(node)
	}

	return nil
}

func unwrap(node *html.Node) {
	parent := node.Parent
	if parent == nil {
		return
	}

	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		node.RemoveChild(child)
		parent.InsertBefore(child, node)
		child = next
	}
	parent.RemoveChild(node)
}

func attr(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}

	return ""
}

func setAttr(node *html.Node, key, value string) {
	for i := range node.Attr {
		if node.Attr[i].Key == key {
			node.Attr[i].Val = value
			return
		}
	}
	node.Attr = append(node.Attr, html.Attribute{Key: key, Val: value})
}

func removeAttr(node *html.Node, key string) {
	attrs := node.Attr[:0]
	for _, attr := range node.Attr {
		if attr.Key != key {
			attrs = append(attrs, attr)
		}
	}
	node.Attr = attrs
}

func addClass(node *html.Node, class string) {
	current := attr(node, "class")
	if current == "" {
		setAttr(node, "class", class)
		return
	}
	for _, existing := range strings.Fields(current) {
		if existing == class {
			return
		}
	}
	setAttr(node, "class", current+" "+class)
}

func renderXHTML(w io.StringWriter, node *html.Node) {
	switch node.Type {
	case html.TextNode:
		writeString(w, html.EscapeString(node.Data))
	case html.ElementNode:
		writeString(w, "<")
		writeString(w, node.Data)
		for _, attr := range node.Attr {
			writeString(w, " ")
			writeString(w, attr.Key)
			writeString(w, `="`)
			writeString(w, html.EscapeString(attr.Val))
			writeString(w, `"`)
		}
		if isVoidElement(node.Data) {
			writeString(w, " />")
			return
		}
		writeString(w, ">")
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			renderXHTML(w, child)
		}
		writeString(w, "</")
		writeString(w, node.Data)
		writeString(w, ">")
	default:
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			renderXHTML(w, child)
		}
	}
}

func writeString(w io.StringWriter, s string) {
	_, _ = w.WriteString(s)
}

func isVoidElement(name string) bool {
	switch name {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "source", "track", "wbr":
		return true
	default:
		return false
	}
}
