package extract

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// AllText mirrors a Scrapy extraction of the form
// response.xpath('//div[@id="mw-content-text"]//text()').extract():
// every text node in MediaWiki's content area, including the text of style
// sheets, citation markers and edit links. Fragments are trimmed, empty ones
// dropped, and the rest joined by single spaces. It exists to show, on the same
// pages, what the cleaner WikipediaArticle extraction removes.
type AllText struct{}

// Extract implements Extractor.
func (AllText) Extract(doc *goquery.Document) (Page, error) {
	content := doc.Find("#mw-content-text").First()
	title := pageTitle(doc)
	if content.Length() == 0 || title == "" {
		return Page{}, ErrNotArticle
	}
	var fragments []string
	for _, node := range content.Nodes {
		fragments = appendTextNodes(fragments, node)
	}
	return Page{Title: title, Text: strings.Join(fragments, " "), RevisionID: revisionID(doc)}, nil
}

// appendTextNodes collects every non-blank text node under node, in document order.
func appendTextNodes(fragments []string, node *html.Node) []string {
	if node.Type == html.TextNode {
		if fragment := normalizeSpace(node.Data); fragment != "" {
			fragments = append(fragments, fragment)
		}
		return fragments
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		fragments = appendTextNodes(fragments, child)
	}
	return fragments
}
