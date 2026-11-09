// Package testsite serves saved pages over HTTP for tests and offline
// benchmarks, so nothing in the test suite touches the live internet.
//
// A request for /wiki/<Name> is answered with <Dir>/<Name>.html, using the same
// file-name rule as scripts/capture-pages.sh. The site can add latency, answer
// chosen paths with an error status, and publish a robots.txt.
package testsite

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Options configure a Site.
type Options struct {
	Dir       string         // directory of saved pages
	Latency   time.Duration  // added before every page response, to simulate the network
	RobotsTxt string         // served at /robots.txt; empty means no robots.txt (404)
	Status    map[string]int // path → HTTP status to answer instead of the page
}

// Site is a running test server.
type Site struct {
	*httptest.Server
	options Options

	inFlight     atomic.Int64
	peakInFlight atomic.Int64
	mu           sync.Mutex
	userAgents   map[string]bool
	pageRequests int
}

// Start serves the saved pages until Close is called.
func Start(options Options) *Site {
	site := &Site{options: options, userAgents: map[string]bool{}}
	site.Server = httptest.NewServer(http.HandlerFunc(site.serve))
	return site
}

// URL returns the site's address for a page path such as "/wiki/Robot".
func (site *Site) URL(path string) string { return site.Server.URL + path }

// PeakInFlight reports the largest number of page requests served at once.
func (site *Site) PeakInFlight() int64 { return site.peakInFlight.Load() }

// PageRequests reports how many page requests (not robots.txt) arrived.
func (site *Site) PageRequests() int {
	site.mu.Lock()
	defer site.mu.Unlock()
	return site.pageRequests
}

// UserAgents returns the distinct User-Agent headers seen on page requests.
func (site *Site) UserAgents() []string {
	site.mu.Lock()
	defer site.mu.Unlock()
	var agents []string
	for agent := range site.userAgents {
		agents = append(agents, agent)
	}
	return agents
}

func (site *Site) serve(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/robots.txt" {
		site.serveRobots(w)
		return
	}
	site.countRequest(r)
	current := site.inFlight.Add(1)
	defer site.inFlight.Add(-1)
	site.recordPeak(current)
	time.Sleep(site.options.Latency)

	if status, ok := site.options.Status[r.URL.Path]; ok {
		http.Error(w, http.StatusText(status), status)
		return
	}
	page, err := os.ReadFile(filepath.Join(site.options.Dir, PageFileName(r.URL.Path)+".html"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(page)
}

func (site *Site) serveRobots(w http.ResponseWriter) {
	if site.options.RobotsTxt == "" {
		http.Error(w, "no robots.txt", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(site.options.RobotsTxt))
}

func (site *Site) countRequest(r *http.Request) {
	site.mu.Lock()
	defer site.mu.Unlock()
	site.pageRequests++
	site.userAgents[r.UserAgent()] = true
}

func (site *Site) recordPeak(current int64) {
	for {
		peak := site.peakInFlight.Load()
		if current <= peak || site.peakInFlight.CompareAndSwap(peak, current) {
			return
		}
	}
}

var awkwardCharacters = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// PageFileName maps a URL path to the saved file's base name, exactly as
// scripts/capture-pages.sh does: the last path segment with awkward characters
// replaced by "_". "/wiki/Android_(robot)" → "Android__robot_".
func PageFileName(path string) string {
	return awkwardCharacters.ReplaceAllString(path[strings.LastIndex(path, "/")+1:], "_")
}
