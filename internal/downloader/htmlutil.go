package downloader

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	xhtml "golang.org/x/net/html"
)

// ParseHTML parses an HTML document.
func ParseHTML(source string) (*xhtml.Node, error) {
	doc, err := xhtml.Parse(strings.NewReader(source))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	return doc, nil
}

// TextContent returns descendant text.
func TextContent(node *xhtml.Node) string {
	var b strings.Builder
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.TextNode {
			_, _ = b.WriteString(n.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)

	return strings.TrimSpace(b.String())
}

// RenderChildren renders a node's children as HTML.
func RenderChildren(node *xhtml.Node) string {
	var b strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		_ = xhtml.Render(&b, child)
	}

	return b.String()
}

// RenderNode renders a node as HTML.
func RenderNode(node *xhtml.Node) string {
	var b bytes.Buffer
	_ = xhtml.Render(&b, node)

	return b.String()
}

// FindAll returns nodes matching predicate in document order.
func FindAll(node *xhtml.Node, match func(*xhtml.Node) bool) []*xhtml.Node {
	var nodes []*xhtml.Node
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if match(n) {
			nodes = append(nodes, n)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)

	return nodes
}

// FindFirst returns the first node matching predicate.
func FindFirst(node *xhtml.Node, match func(*xhtml.Node) bool) *xhtml.Node {
	var found *xhtml.Node
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if found != nil {
			return
		}
		if match(n) {
			found = n
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)

	return found
}

// Attr returns an attribute value.
func Attr(node *xhtml.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}

	return ""
}

// HasClass checks a whitespace-separated class attribute.
func HasClass(node *xhtml.Node, class string) bool {
	for _, field := range strings.Fields(Attr(node, "class")) {
		if field == class {
			return true
		}
	}

	return false
}

// IsElement checks an element name.
func IsElement(name string) func(*xhtml.Node) bool {
	return func(node *xhtml.Node) bool {
		return node.Type == xhtml.ElementNode && node.Data == name
	}
}

// ReadAllString is a test/helper bridge for sources that expose readers.
func ReadAllString(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
