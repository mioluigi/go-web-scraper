package extract

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Block elements start and end a line of text. Without this, the text of two
// adjacent blocks runs together ("EngineeringComputer science"), which is what
// goquery's Text() does (spike 001, §3).
var blockElements = map[atom.Atom]bool{
	atom.Address: true, atom.Article: true, atom.Aside: true, atom.Blockquote: true,
	atom.Br: true, atom.Dd: true, atom.Div: true, atom.Dl: true, atom.Dt: true,
	atom.Figcaption: true, atom.Figure: true, atom.Footer: true, atom.Form: true,
	atom.H1: true, atom.H2: true, atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true,
	atom.Header: true, atom.Hr: true, atom.Li: true, atom.Main: true, atom.Nav: true,
	atom.Ol: true, atom.P: true, atom.Pre: true, atom.Section: true, atom.Table: true,
	atom.Tr: true, atom.Ul: true,
}

// Table cells are separated by a space, so neighbouring cells stay separate words.
var cellElements = map[atom.Atom]bool{atom.Td: true, atom.Th: true}

// Elements whose content is code or styling, never readable text.
var nonTextElements = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Noscript: true, atom.Template: true,
}

// BlockText returns the readable text of a selection: one line per block
// element, white space collapsed inside each line, empty lines dropped, and
// scripts and styles ignored. It works for any site's HTML.
func BlockText(selection *goquery.Selection) string {
	var text strings.Builder
	for _, node := range selection.Nodes {
		writeText(&text, node)
	}
	return tidyLines(text.String())
}

// writeText appends the text under node, marking block boundaries with newlines.
func writeText(text *strings.Builder, node *html.Node) {
	if node.Type == html.TextNode {
		text.WriteString(node.Data)
		return
	}
	isElement := node.Type == html.ElementNode
	if isElement && nonTextElements[node.DataAtom] {
		return
	}
	separator := ""
	if isElement && blockElements[node.DataAtom] {
		separator = "\n"
	} else if isElement && cellElements[node.DataAtom] {
		separator = " "
	}
	text.WriteString(separator)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		writeText(text, child)
	}
	text.WriteString(separator)
}

// tidyLines collapses white space within each line and drops the empty lines.
func tidyLines(raw string) string {
	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		if tidy := normalizeSpace(line); tidy != "" {
			lines = append(lines, tidy)
		}
	}
	return strings.Join(lines, "\n")
}
