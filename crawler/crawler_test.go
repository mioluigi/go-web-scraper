package crawler

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mioluigi/go-web-scraper/extract"
	"github.com/mioluigi/go-web-scraper/internal/testsite"
)

// syntheticSite serves count copies of the synthetic article as /wiki/Page_0 …
// plus a page that is not an article, at /wiki/Not_an_article.
func syntheticSite(t testing.TB, count int, options testsite.Options) (*testsite.Site, []string) {
	t.Helper()
	article, err := os.ReadFile("../extract/testdata/article.html")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	var paths []string
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("Page_%d", i)
		// Give each copy its own title so records differ.
		page := strings.Replace(string(article), `<span class="mw-page-title-main">Robotics</span>`, `<span>`+name+`</span>`, 1)
		if err := os.WriteFile(filepath.Join(dir, name+".html"), []byte(page), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, "/wiki/"+name)
	}
	os.WriteFile(filepath.Join(dir, "Not_an_article.html"), []byte("<html><body><p>Hello</p></body></html>"), 0o644)
	options.Dir = dir
	site := testsite.Start(options)
	t.Cleanup(site.Close)
	return site, paths
}

// savedPagesSite serves the real pages captured by scripts/capture-pages.sh.
func savedPagesSite(t testing.TB, options testsite.Options) (*testsite.Site, []string) {
	t.Helper()
	files, _ := filepath.Glob("../testdata/wikipedia/*.html")
	if len(files) == 0 {
		t.Skip("no saved pages yet: run scripts/capture-pages.sh")
	}
	options.Dir = "../testdata/wikipedia"
	site := testsite.Start(options)
	t.Cleanup(site.Close)
	var paths []string
	for _, file := range files {
		paths = append(paths, "/wiki/"+strings.TrimSuffix(filepath.Base(file), ".html"))
	}
	return site, paths
}

func urlsOn(site *testsite.Site, paths ...string) []string {
	urls := make([]string, len(paths))
	for i, path := range paths {
		urls[i] = site.URL(path)
	}
	return urls
}

var testUserAgent = UserAgent("tests@example.org")

func testConfig(parallelism int) Config {
	config := DefaultConfig()
	config.UserAgent = testUserAgent
	config.Parallelism = parallelism
	return config
}

// withoutTimes removes the one field that legitimately differs between runs.
func withoutTimes(results []Result) []Result {
	copied := append([]Result(nil), results...)
	for i := range copied {
		copied[i].Page.RetrievedAt = time.Time{}
	}
	return copied
}

func TestResultsComeBackInInputOrderWithPages(t *testing.T) {
	site, paths := syntheticSite(t, 6, testsite.Options{})
	urls := urlsOn(site, paths...)
	results, err := Crawl(urls, testConfig(4), extract.WikipediaArticle{})
	if err != nil {
		t.Fatal(err)
	}
	for i, result := range results {
		if !result.OK() {
			t.Fatalf("%s: %v", urls[i], result.Err)
		}
		if result.Page.URL != urls[i] || result.Page.Title != fmt.Sprintf("Page_%d", i) {
			t.Errorf("result %d = %s %q, want %s Page_%d", i, result.Page.URL, result.Page.Title, urls[i], i)
		}
		if result.Status != 200 || result.Page.WordCount == 0 || result.Page.RetrievedAt.IsZero() {
			t.Errorf("result %d incomplete: %+v", i, result)
		}
	}
}

// AC-2 / Constitution 1.1: concurrency must not change what is scraped.
func TestConcurrentCrawlEqualsSequentialCrawl(t *testing.T) {
	for _, extractor := range []extract.Extractor{extract.WikipediaArticle{}, extract.AllText{}} {
		site, paths := savedPagesSite(t, testsite.Options{})
		urls := urlsOn(site, paths...)
		sequential, err := Crawl(urls, testConfig(1), extractor)
		if err != nil {
			t.Fatal(err)
		}
		concurrent, err := Crawl(urls, testConfig(8), extractor)
		if err != nil {
			t.Fatal(err)
		}
		for i := range urls {
			if !sequential[i].OK() {
				t.Fatalf("%s: %v", urls[i], sequential[i].Err)
			}
			if a, b := withoutTimes(sequential)[i], withoutTimes(concurrent)[i]; a.Page != b.Page || a.Status != b.Status {
				t.Errorf("%T, %s: sequential and concurrent records differ", extractor, paths[i])
			}
		}
	}
}

func TestParallelismIsBoundedPerHost(t *testing.T) {
	for _, parallelism := range []int{1, 4} {
		site, paths := syntheticSite(t, 12, testsite.Options{Latency: 30 * time.Millisecond})
		if _, err := Crawl(urlsOn(site, paths...), testConfig(parallelism), extract.WikipediaArticle{}); err != nil {
			t.Fatal(err)
		}
		if peak := site.PeakInFlight(); peak != int64(parallelism) {
			t.Errorf("parallelism %d: peak requests in flight = %d", parallelism, peak)
		}
	}
}

func TestOneBadPageDoesNotStopTheOthers(t *testing.T) {
	site, paths := syntheticSite(t, 3, testsite.Options{Status: map[string]int{"/wiki/Page_1": 500}})
	urls := urlsOn(site, append(paths, "/wiki/Missing", "/wiki/Not_an_article")...)
	results, err := Crawl(urls, testConfig(4), extract.WikipediaArticle{})
	if err != nil {
		t.Fatal(err)
	}
	wantErrors := map[int]string{1: "HTTP 500 Internal Server Error", 3: "HTTP 404 Not Found", 4: extract.ErrNotArticle.Error()}
	for i, result := range results {
		want, shouldFail := wantErrors[i]
		if !shouldFail && !result.OK() {
			t.Errorf("%s failed: %v", urls[i], result.Err)
		}
		if shouldFail && (result.OK() || result.Err.Error() != want) {
			t.Errorf("%s: err = %v, want %q", urls[i], result.Err, want)
		}
		if result.Page.URL != urls[i] {
			t.Errorf("result %d names %s", i, result.Page.URL)
		}
	}
}

// AC-6: robots.txt is obeyed and the crawler identifies itself.
func TestRobotsTxtIsObeyedAndUserAgentIsSent(t *testing.T) {
	site, paths := syntheticSite(t, 2, testsite.Options{RobotsTxt: "User-agent: *\nDisallow: /wiki/Page_1\n"})
	results, err := Crawl(urlsOn(site, paths...), testConfig(2), extract.WikipediaArticle{})
	if err != nil {
		t.Fatal(err)
	}
	if !results[0].OK() {
		t.Errorf("allowed page failed: %v", results[0].Err)
	}
	if results[1].OK() || !strings.Contains(results[1].Err.Error(), "robots.txt disallows") {
		t.Errorf("disallowed page: err = %v, want a robots.txt refusal", results[1].Err)
	}
	if site.PageRequests() != 1 {
		t.Errorf("site received %d page requests, want 1 (the disallowed page must not be fetched)", site.PageRequests())
	}
	agents := site.UserAgents()
	if len(agents) != 1 || agents[0] != "go-web-scraper/1.0 (tests@example.org)" {
		t.Errorf("User-Agents seen = %q, want only the form Wikipedia accepted", agents)
	}
}

func TestValidContact(t *testing.T) {
	for contact, want := range map[string]bool{
		"luis@example.org": true, "a.b+c@gids.co": true,
		"": false, "luis": false, "luis@localhost": false, "https://github.com/x": false,
		"a@b.c (x)": false, "a@b.c; d@e.f": false, "two words@x.org": false,
	} {
		if got := ValidContact(contact); got != want {
			t.Errorf("ValidContact(%q) = %v, want %v", contact, got, want)
		}
	}
}

func TestIgnoreRobotsFetchesDisallowedPages(t *testing.T) {
	site, paths := syntheticSite(t, 1, testsite.Options{RobotsTxt: "User-agent: *\nDisallow: /\n"})
	config := testConfig(1)
	config.IgnoreRobots = true
	results, _ := Crawl(urlsOn(site, paths...), config, extract.WikipediaArticle{})
	if !results[0].OK() {
		t.Errorf("err = %v, want the page fetched", results[0].Err)
	}
}

func TestSlowPagesTimeOut(t *testing.T) {
	site, paths := syntheticSite(t, 1, testsite.Options{Latency: 300 * time.Millisecond})
	config := testConfig(1)
	config.Timeout = 50 * time.Millisecond
	results, _ := Crawl(urlsOn(site, paths...), config, extract.WikipediaArticle{})
	if results[0].OK() || !strings.HasPrefix(results[0].Err.Error(), "no response") {
		t.Errorf("err = %v, want a no-response timeout", results[0].Err)
	}
}

func TestOnResultIsCalledOncePerURL(t *testing.T) {
	site, paths := syntheticSite(t, 5, testsite.Options{})
	seen := map[int]int{}
	config := testConfig(3)
	config.OnResult = func(index int, _ Result) { seen[index]++ } // safe: calls never overlap
	Crawl(urlsOn(site, append(paths, "/wiki/Missing")...), config, extract.WikipediaArticle{})
	if len(seen) != 6 {
		t.Errorf("OnResult saw %d URLs, want 6", len(seen))
	}
	for index, calls := range seen {
		if calls != 1 {
			t.Errorf("URL %d reported %d times", index, calls)
		}
	}
}

func TestCrawlRejectsUnusableSettings(t *testing.T) {
	noUserAgent := testConfig(4)
	noUserAgent.UserAgent = " "
	cases := []struct {
		urls   []string
		config Config
		want   error
	}{
		{nil, testConfig(4), ErrNoURLs},
		{[]string{"http://example.org/"}, DefaultConfig(), ErrMissingUserAgent},
		{[]string{"http://example.org/"}, testConfig(0), ErrBadParallelism},
		{[]string{"http://example.org/"}, noUserAgent, ErrMissingUserAgent},
	}
	for _, c := range cases {
		if _, err := Crawl(c.urls, c.config, extract.WikipediaArticle{}); !errors.Is(err, c.want) {
			t.Errorf("Crawl(%v) err = %v, want %v", c.urls, err, c.want)
		}
	}
	if _, err := Crawl([]string{"not a url"}, testConfig(4), extract.WikipediaArticle{}); err == nil {
		t.Error("want an error for a URL without a host")
	}
}

func TestIsHTML(t *testing.T) {
	for contentType, want := range map[string]bool{
		"": true, "text/html": true, "text/html; charset=utf-8": true, "application/xhtml+xml": true,
		"application/json": false, "image/png": false, "text/html; ===": false,
	} {
		if got := isHTML(contentType); got != want {
			t.Errorf("isHTML(%q) = %v, want %v", contentType, got, want)
		}
	}
}

// AC-7 / NFR-2: no goroutine outlives a crawl, with or without failures.
func TestNoGoroutineLeaks(t *testing.T) {
	site, paths := syntheticSite(t, 8, testsite.Options{
		Latency: 5 * time.Millisecond, Status: map[string]int{"/wiki/Page_2": 503},
		RobotsTxt: "User-agent: *\nDisallow: /wiki/Page_3\n",
	})
	urls := urlsOn(site, append(paths, "/wiki/Missing")...)
	before := runtime.NumGoroutine()
	for run := 0; run < 20; run++ {
		if _, err := Crawl(urls, testConfig(4), extract.WikipediaArticle{}); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if after := runtime.NumGoroutine(); after > before {
		t.Errorf("goroutines: %d before, %d after 20 crawls", before, after)
	}
}

// Offline throughput benchmark (NFR-4a): saved pages, 100 ms simulated latency.
func BenchmarkCrawlSavedPages(b *testing.B) {
	for _, parallelism := range []int{1, 2, 4, 8} {
		b.Run(fmt.Sprintf("parallel=%d", parallelism), func(b *testing.B) {
			site, paths := savedPagesSite(b, testsite.Options{Latency: 100 * time.Millisecond})
			urls := urlsOn(site, paths...)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := Crawl(urls, testConfig(parallelism), extract.WikipediaArticle{}); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(b.N*len(urls))/b.Elapsed().Seconds(), "pages/s")
		})
	}
}
