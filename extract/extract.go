// Package extract turns one HTML document into one Page: a title, the readable
// text, and a few facts that let a knowledge base trace the text to its source.
//
// The site-specific part of a scraper lives here, behind the Extractor
// interface: the crawler fetches pages for any site, and an Extractor decides
// what to keep. WikipediaArticle keeps an article's prose; AllText reproduces a
// Scrapy-style "every text node in the content area" extraction for comparison.
package extract

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Page is one scraped page, as written to one line of the JSON Lines file.
// URL and RetrievedAt are filled in by the crawler; the rest by an Extractor.
type Page struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Text        string    `json:"text"`
	WordCount   int       `json:"word_count"`
	RevisionID  int64     `json:"revision_id,omitempty"`
	RetrievedAt time.Time `json:"retrieved_at"`
}

// Extractor reads the parts of a document that a particular site keeps in a
// particular place. Implementations must not modify doc: the same document may
// be read by more than one extractor.
type Extractor interface {
	Extract(doc *goquery.Document) (Page, error)
}

// ErrNotArticle reports a page without the expected title or content area,
// for example an error page or a page from another site.
var ErrNotArticle = errors.New("page has no article title or content area")

// ByName returns the extractor selected by a command-line name.
func ByName(name string) (Extractor, bool) {
	switch name {
	case "wikipedia":
		return WikipediaArticle{}, true
	case "alltext":
		return AllText{}, true
	}
	return nil, false
}

// Names lists the values ByName accepts, for help and error messages.
func Names() []string { return []string{"wikipedia", "alltext"} }

// MediaWiki pages announce their revision in a configuration script.
var revisionPattern = regexp.MustCompile(`"wgRevisionId":\s*(\d+)`)

// revisionID returns the MediaWiki revision of the page, or 0 if there is none.
func revisionID(doc *goquery.Document) int64 {
	var revision int64
	doc.Find("script").EachWithBreak(func(_ int, script *goquery.Selection) bool {
		match := revisionPattern.FindStringSubmatch(script.Text())
		if match == nil {
			return true
		}
		revision, _ = strconv.ParseInt(match[1], 10, 64) // digits only: cannot fail short of overflow
		return false
	})
	return revision
}

// normalizeSpace collapses every run of white space to one space and trims the ends.
func normalizeSpace(s string) string { return strings.Join(strings.Fields(s), " ") }

// pageTitle reads MediaWiki's main heading.
func pageTitle(doc *goquery.Document) string {
	return normalizeSpace(doc.Find("h1#firstHeading").First().Text())
}
