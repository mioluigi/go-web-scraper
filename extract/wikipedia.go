package extract

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// WikipediaArticle keeps the prose of a Wikipedia article: paragraphs, lists
// and section headings, one per line. It drops what a reader of the knowledge
// base would not want as text: styles, citation markers, edit links, infoboxes,
// navigation boxes, tables, images and captions, mathematical markup (replaced
// by its plain-text form), and the back matter (See also, References, External
// links and similar sections).
type WikipediaArticle struct{}

// contentSelector is MediaWiki's container for the rendered article.
const contentSelector = "#mw-content-text .mw-parser-output"

// noiseSelector lists the parts of a MediaWiki article that are not prose.
var noiseSelector = strings.Join([]string{
	"style", "script", "noscript", "link", "meta",
	"sup.reference", "sup.noprint", ".mw-editsection", ".mw-cite-backlink",
	".hatnote", ".shortdescription", ".ambox", ".ombox", ".metadata", ".noprint", ".mw-empty-elt",
	".infobox", ".sidebar", ".navbox", ".navbox-styles", ".vertical-navbox", ".portalbox",
	".side-box", ".sistersitebox", ".mw-authority-control", ".navigation-not-searchable",
	".reflist", ".references", ".mw-references-wrap", ".refbegin",
	"table", "figure", ".thumb", ".gallery", ".toc", "#toc",
}, ", ")

// backMatterHeadings name the closing sections that hold links and citations
// rather than prose; everything from such a heading to the next top-level
// heading is dropped.
var backMatterHeadings = map[string]bool{
	"see also": true, "references": true, "notes": true, "footnotes": true,
	"citations": true, "sources": true, "bibliography": true, "further reading": true,
	"external links": true, "notes and references": true, "works cited": true,
	"general references": true, "general and cited references": true,
}

// Extract implements Extractor.
func (WikipediaArticle) Extract(doc *goquery.Document) (Page, error) {
	content := doc.Find(contentSelector).First()
	title := pageTitle(doc)
	if content.Length() == 0 || title == "" {
		return Page{}, ErrNotArticle
	}
	article := content.Clone() // Extract must not change doc
	replaceMathWithPlainText(article)
	article.Find(noiseSelector).Remove()
	removeBackMatter(article)
	return Page{Title: title, Text: BlockText(article), RevisionID: revisionID(doc)}, nil
}

// replaceMathWithPlainText swaps each formula's MathML and fallback image for
// the TeX source MediaWiki keeps in the alttext attribute, e.g. "x^{2}".
func replaceMathWithPlainText(article *goquery.Selection) {
	article.Find(".mwe-math-element").Each(func(_ int, formula *goquery.Selection) {
		tex := formula.Find("math").AttrOr("alttext", "")
		tex = strings.TrimPrefix(strings.TrimSpace(tex), `{\displaystyle`)
		tex = strings.TrimSpace(strings.TrimSuffix(tex, "}"))
		formula.ReplaceWithHtml(" " + htmlEscaper.Replace(tex) + " ")
	})
}

var htmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// removeBackMatter deletes back-matter sections from the article's top level.
// MediaWiki marks a top-level heading as <div class="mw-heading mw-heading2">
// around an <h2> (current markup) or as a bare <h2> (older markup).
func removeBackMatter(article *goquery.Selection) {
	inBackMatter := false
	article.Children().Each(func(_ int, block *goquery.Selection) {
		if isTopLevelHeading(block) {
			inBackMatter = backMatterHeadings[strings.ToLower(normalizeSpace(block.Text()))]
		}
		if inBackMatter {
			block.Remove()
		}
	})
}

func isTopLevelHeading(block *goquery.Selection) bool {
	return block.Is("h2") || block.Is("div.mw-heading2")
}
