package extract

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func loadDocument(t testing.TB, path string) *goquery.Document {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer file.Close()
	doc, err := goquery.NewDocumentFromReader(file)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return doc
}

func documentFromString(t *testing.T, markup string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(markup))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return doc
}

// The expected text is written by hand from the fixture, not produced by the code.
const wantArticleText = `Robotics is the interdisciplinary study & practice of the design of robots. It combines several fields.
History
Early automata date to antiquity.
Mechanical engineering
Computer science
Learning
The reward is r_{t}<1 per step.`

// The same article in both shapes of MediaWiki markup: flat (headings and
// content as siblings) and sectioned (each section in a <section>), the shape
// the live pages use. Both must give exactly the same text.
func TestWikipediaArticleKeepsProseOnly(t *testing.T) {
	for _, fixture := range []string{"testdata/article.html", "testdata/article-sections.html"} {
		page, err := WikipediaArticle{}.Extract(loadDocument(t, fixture))
		if err != nil {
			t.Fatalf("%s: Extract: %v", fixture, err)
		}
		if page.Title != "Robotics" {
			t.Errorf("%s: Title = %q, want %q", fixture, page.Title, "Robotics")
		}
		if page.RevisionID != 1254321987 {
			t.Errorf("%s: RevisionID = %d, want 1254321987", fixture, page.RevisionID)
		}
		if page.Text != wantArticleText {
			t.Errorf("%s: Text mismatch\n got:\n%s\nwant:\n%s", fixture, page.Text, wantArticleText)
		}
	}
}

func TestWikipediaArticleDoesNotChangeTheDocument(t *testing.T) {
	doc := loadDocument(t, "testdata/article.html")
	before, _ := doc.Html()
	if _, err := (WikipediaArticle{}).Extract(doc); err != nil {
		t.Fatal(err)
	}
	after, _ := doc.Html()
	if before != after {
		t.Error("Extract modified the document it was given")
	}
}

func TestAllTextKeepsEverythingLikeScrapy(t *testing.T) {
	page, err := AllText{}.Extract(loadDocument(t, "testdata/article.html"))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	for _, noise := range []string{".mw-parser-output .hatnote{font-style:italic}", "[1]", "edit", "Smith, J. (2020)"} {
		if !strings.Contains(page.Text, noise) {
			t.Errorf("AllText should keep %q, as Scrapy's //text() does", noise)
		}
	}
	if strings.Contains(page.Text, "Main page") || strings.Contains(page.Text, "Creative Commons") {
		t.Error("AllText must stay inside #mw-content-text")
	}
}

func TestExtractorsRejectPagesThatAreNotArticles(t *testing.T) {
	cases := map[string]string{
		"no content area": `<html><body><h1 id="firstHeading">Title</h1><p>text</p></body></html>`,
		"no title":        `<html><body><div id="mw-content-text"><div class="mw-parser-output"><p>x</p></div></div></body></html>`,
	}
	for name, markup := range cases {
		for _, extractor := range []Extractor{WikipediaArticle{}, AllText{}} {
			if _, err := extractor.Extract(documentFromString(t, markup)); err != ErrNotArticle {
				t.Errorf("%s, %T: err = %v, want ErrNotArticle", name, extractor, err)
			}
		}
	}
}

func TestBlockTextSeparatesBlocksAndCells(t *testing.T) {
	doc := documentFromString(t, `<div><p>One  <b>two</b></p><ul><li>three</li><li>four</li></ul>`+
		`<table><tr><td>five</td><td>six</td></tr></table>five<br>seven<script>x()</script></div>`)
	want := "One two\nthree\nfour\nfive six\nfive\nseven"
	if got := BlockText(doc.Find("div")); got != want {
		t.Errorf("BlockText = %q, want %q", got, want)
	}
}

func TestRevisionIDIsZeroWhenAbsent(t *testing.T) {
	if got := revisionID(documentFromString(t, `<html><script>var x = 1;</script></html>`)); got != 0 {
		t.Errorf("revisionID = %d, want 0", got)
	}
}

func TestByName(t *testing.T) {
	for _, name := range Names() {
		if _, ok := ByName(name); !ok {
			t.Errorf("ByName(%q) not found", name)
		}
	}
	if _, ok := ByName("scrapy"); ok {
		t.Error("ByName accepted an unknown name")
	}
}

// Quality checks on the real pages saved by scripts/capture-pages.sh (AC-3).
var (
	citationMarker = regexp.MustCompile(`\[\d+\]`)
	cssRule        = regexp.MustCompile(`[.#][\w-]+[^{}\n]*\{[^}]*\}`)
)

func TestWikipediaArticleOnSavedPages(t *testing.T) {
	pages, _ := filepath.Glob("../testdata/wikipedia/*.html")
	if len(pages) == 0 {
		t.Skip("no saved pages yet: run scripts/capture-pages.sh (acceptance.sh requires them)")
	}
	for _, path := range pages {
		t.Run(filepath.Base(path), func(t *testing.T) {
			page, err := WikipediaArticle{}.Extract(loadDocument(t, path))
			if err != nil {
				t.Fatalf("Extract: %v", err)
			}
			wantTitle := strings.ReplaceAll(strings.TrimSuffix(filepath.Base(path), ".html"), "_", " ")
			if normalizeSpace(strings.Map(keepAlphanumeric, page.Title)) != normalizeSpace(strings.Map(keepAlphanumeric, wantTitle)) {
				t.Errorf("Title = %q, file suggests %q", page.Title, wantTitle)
			}
			if len(strings.Fields(page.Text)) < 300 {
				t.Errorf("only %d words of text", len(strings.Fields(page.Text)))
			}
			if page.RevisionID == 0 {
				t.Error("no revision ID")
			}
			for name, pattern := range map[string]*regexp.Regexp{"citation marker": citationMarker, "CSS rule": cssRule} {
				if found := pattern.FindString(page.Text); found != "" {
					t.Errorf("text contains a %s: %q", name, found)
				}
			}
			if strings.Contains(page.Text, "[edit]") {
				t.Error("text contains [edit]")
			}
			for _, line := range strings.Split(page.Text, "\n") {
				if backMatterHeadings[strings.ToLower(line)] {
					t.Errorf("back-matter section kept: %q", line)
				}
			}
			if glued := gluedBlockPairs(t, path, page.Text); len(glued) > 0 {
				t.Errorf("%d neighbouring blocks run together, e.g. %q", len(glued), glued[0])
			}
		})
	}
}

// gluedBlockPairs looks at every pair of neighbouring list items or
// paragraphs in the page and reports those whose boundary words appear joined
// in the text ("engineering"+"Computer" → "engineeringComputer"). Checking the
// actual boundaries avoids flagging real names such as "iRobot" or "AlphaGo",
// which a capital-letter pattern cannot tell apart (lesson L-W7-12).
func gluedBlockPairs(t *testing.T, path, text string) []string {
	doc := loadDocument(t, path)
	var glued []string
	doc.Find(contentSelector + " li + li, " + contentSelector + " p + p").Each(func(_ int, block *goquery.Selection) {
		previous := strings.Fields(block.Prev().Text())
		current := strings.Fields(block.Text())
		if len(previous) == 0 || len(current) == 0 {
			return
		}
		joined := previous[len(previous)-1] + current[0]
		if len(joined) > 6 && strings.Contains(text, joined) {
			glued = append(glued, joined)
		}
	})
	return glued
}

// keepAlphanumeric maps punctuation to spaces, so "Android (robot)" and the
// file name "Android__robot_" compare equal.
func keepAlphanumeric(r rune) rune {
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
		return r
	}
	return ' '
}

func BenchmarkWikipediaArticle(b *testing.B) {
	pages, _ := filepath.Glob("../testdata/wikipedia/*.html")
	if len(pages) == 0 {
		pages = []string{"testdata/article.html"}
	}
	docs := make([]*goquery.Document, len(pages))
	for i, path := range pages {
		docs[i] = loadDocument(b, path)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, doc := range docs {
			if _, err := (WikipediaArticle{}).Extract(doc); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.ReportMetric(float64(b.N*len(docs))/b.Elapsed().Seconds(), "pages/s")
}
