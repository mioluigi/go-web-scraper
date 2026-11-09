// Package crawler fetches a list of pages concurrently with Colly and turns
// each one into an extract.Page, using an extract.Extractor chosen by the
// caller. It knows nothing about any particular site, prints nothing and never
// exits the program: give it URLs, a Config and an Extractor, and it returns
// one Result per URL, in the order of the URLs.
//
// The crawler is polite by default (Constitution, Article 9): it requires a
// User-Agent that identifies whoever runs it, obeys robots.txt, stays on the hosts of the given URLs, and limits
// how many requests are in flight to each host.
package crawler

import (
	"bytes"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"

	"github.com/mioluigi/go-web-scraper/extract"
)

// Version is the scraper's version, part of its User-Agent.
const Version = "1.0"

// UserAgent identifies the crawler and how to reach the person running it, in
// the form Wikimedia accepts: "go-web-scraper/1.0 (name@example.org)".
//
// Measured on 2026-11-08 (spike 001, addendum): Wikipedia answered 403 to every
// User-Agent that carried this repository's github.com URL, with or without an
// e-mail address, and 200 to the e-mail address alone. The contact is therefore
// an e-mail address supplied at run time and never stored in the repository
// (Constitution 8.2).
func UserAgent(contactEmail string) string {
	return fmt.Sprintf("go-web-scraper/%s (%s)", Version, contactEmail)
}

var emailPattern = regexp.MustCompile(`^[^@\s();]+@[^@\s();]+\.[^@\s();]+$`)

// ValidContact reports whether contactEmail looks like an e-mail address that
// can go inside the User-Agent's parentheses.
func ValidContact(contactEmail string) bool { return emailPattern.MatchString(contactEmail) }

// Config controls one crawl. Start from DefaultConfig.
type Config struct {
	Parallelism  int           // most requests in flight per host; 1 = sequential
	Delay        time.Duration // pause before each new request to a host
	Timeout      time.Duration // limit for one request, including reading the body
	UserAgent    string        // identifies the crawler to the site
	CacheDir     string        // if set, reuse responses saved here (development only)
	IgnoreRobots bool          // false (the default) obeys robots.txt

	// OnResult, if set, is called once per URL as soon as its result is known
	// (for progress messages). Calls never overlap, but they come from the
	// crawler's goroutines, in completion order.
	OnResult func(index int, result Result)
}

// DefaultConfig returns the settings the command-line program starts from.
// UserAgent is left empty on purpose: the caller must say who runs the crawl.
func DefaultConfig() Config {
	return Config{Parallelism: 4, Timeout: 30 * time.Second}
}

// Result is the outcome for one URL: a Page, or an error saying why not.
type Result struct {
	Page   extract.Page // Page.URL is always the requested URL
	Status int          // HTTP status, 0 if no response arrived
	Err    error
}

// OK reports whether the page was fetched and extracted.
func (result Result) OK() bool { return result.Err == nil }

// Errors returned by Crawl for unusable settings.
var (
	ErrNoURLs           = errors.New("crawler: no URLs to crawl")
	ErrBadParallelism   = errors.New("crawler: parallelism must be at least 1")
	ErrMissingUserAgent = errors.New("crawler: a User-Agent is required (see UserAgent)")
)

// indexKey stores a request's position in the URL list in Colly's request context.
const indexKey = "index"

// Crawl fetches every URL and extracts a Page from each. One failing URL never
// stops the others; its Result carries the error. Crawl returns when every
// request has finished, and leaves no goroutine or idle connection behind.
func Crawl(urls []string, config Config, extractor extract.Extractor) ([]Result, error) {
	if err := validate(urls, config); err != nil {
		return nil, err
	}
	hosts, err := hostsOf(urls)
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	collector, err := newCollector(config, hosts, transport)
	if err != nil {
		return nil, err
	}
	results := newResults(urls)
	var mu sync.Mutex // one writer for results, as Colly callbacks run concurrently
	record := func(index int, update func(*Result)) {
		mu.Lock()
		defer mu.Unlock()
		update(&results[index])
		if config.OnResult != nil {
			config.OnResult(index, results[index])
		}
	}

	collector.OnResponse(func(response *colly.Response) {
		index := response.Ctx.GetAny(indexKey).(int)
		page, err := pageFrom(response, extractor)
		record(index, func(result *Result) {
			result.Status = response.StatusCode
			if err != nil {
				result.Err = err
				return
			}
			page.URL = result.Page.URL
			result.Page = page
		})
	})
	collector.OnError(func(response *colly.Response, err error) {
		index := response.Ctx.GetAny(indexKey).(int)
		record(index, func(result *Result) {
			result.Status = response.StatusCode
			result.Err = describeFetchError(response.StatusCode, err)
		})
	})

	for index, pageURL := range urls {
		context := colly.NewContext()
		context.Put(indexKey, index)
		// Checks that fail before any request (robots.txt, host not allowed)
		// are returned here rather than through OnError.
		if err := collector.Request(http.MethodGet, pageURL, nil, context, nil); err != nil {
			record(index, func(result *Result) { result.Err = describeRefusal(err) })
		}
	}
	collector.Wait()
	transport.CloseIdleConnections() // keep-alive connections would otherwise hold goroutines
	return results, nil
}

func validate(urls []string, config Config) error {
	if len(urls) == 0 {
		return ErrNoURLs
	}
	if config.Parallelism < 1 {
		return ErrBadParallelism
	}
	if strings.TrimSpace(config.UserAgent) == "" {
		return ErrMissingUserAgent
	}
	return nil
}

// hostsOf returns the distinct hosts of urls; the crawl is confined to them.
func hostsOf(urls []string) ([]string, error) {
	seen := map[string]bool{}
	var hosts []string
	for _, raw := range urls {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Hostname() == "" {
			return nil, fmt.Errorf("crawler: unusable URL %q", raw)
		}
		if !seen[parsed.Hostname()] {
			seen[parsed.Hostname()] = true
			hosts = append(hosts, parsed.Hostname())
		}
	}
	return hosts, nil
}

func newCollector(config Config, hosts []string, transport *http.Transport) (*colly.Collector, error) {
	options := []colly.CollectorOption{
		colly.Async(true),
		colly.UserAgent(config.UserAgent),
		colly.AllowedDomains(hosts...),
	}
	if config.CacheDir != "" {
		options = append(options, colly.CacheDir(config.CacheDir))
	}
	collector := colly.NewCollector(options...)
	collector.IgnoreRobotsTxt = config.IgnoreRobots // Colly's own default is to ignore robots.txt (spike 001 §2)
	collector.WithTransport(transport)
	if config.Timeout > 0 {
		collector.SetRequestTimeout(config.Timeout)
	}
	limit := &colly.LimitRule{DomainGlob: "*", Parallelism: config.Parallelism, Delay: config.Delay}
	if err := collector.Limit(limit); err != nil {
		return nil, fmt.Errorf("crawler: %w", err)
	}
	return collector, nil
}

// newResults prepares one Result per URL, each already naming its URL.
func newResults(urls []string) []Result {
	results := make([]Result, len(urls))
	for index, pageURL := range urls {
		results[index].Page.URL = pageURL
	}
	return results
}

// pageFrom parses an HTML response and applies the extractor.
func pageFrom(response *colly.Response, extractor extract.Extractor) (extract.Page, error) {
	if !isHTML(response.Headers.Get("Content-Type")) {
		return extract.Page{}, fmt.Errorf("not an HTML page (Content-Type %q)", response.Headers.Get("Content-Type"))
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(response.Body))
	if err != nil {
		return extract.Page{}, fmt.Errorf("parse HTML: %w", err)
	}
	page, err := extractor.Extract(doc)
	if err != nil {
		return extract.Page{}, err
	}
	page.WordCount = len(strings.Fields(page.Text))
	page.RetrievedAt = time.Now().UTC().Truncate(time.Second)
	return page, nil
}

// isHTML accepts a missing Content-Type (some servers omit it) or an HTML one.
func isHTML(contentType string) bool {
	if contentType == "" {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && (mediaType == "text/html" || mediaType == "application/xhtml+xml")
}

// describeFetchError turns Colly's terse errors into messages for people.
func describeFetchError(status int, err error) error {
	if status > 0 {
		return fmt.Errorf("HTTP %d %s", status, http.StatusText(status))
	}
	return fmt.Errorf("no response: %w", err)
}

// describeRefusal explains a request Colly refused to send.
func describeRefusal(err error) error {
	switch {
	case errors.Is(err, colly.ErrRobotsTxtBlocked):
		return fmt.Errorf("not fetched: the site's robots.txt disallows it (%w)", err)
	case errors.Is(err, colly.ErrForbiddenDomain):
		return fmt.Errorf("not fetched: host outside the crawl (%w)", err)
	}
	return fmt.Errorf("not fetched: %w", err)
}
