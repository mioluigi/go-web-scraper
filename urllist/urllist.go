// Package urllist reads a list of URLs to crawl: one absolute http or https
// URL per line, with blank lines and lines starting with "#" ignored.
package urllist

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
)

// List is a validated list of URLs, in input order, without duplicates.
type List struct {
	URLs       []string // each URL once, in the order first seen
	Duplicates []string // URLs that appeared more than once (reported, not crawled twice)
}

// Hosts returns each distinct host in the list, in order of first use.
// The crawler is confined to these hosts.
func (list List) Hosts() []string {
	seen := map[string]bool{}
	var hosts []string
	for _, raw := range list.URLs {
		host := mustParse(raw).Hostname()
		if !seen[host] {
			seen[host] = true
			hosts = append(hosts, host)
		}
	}
	return hosts
}

// LineError describes one unusable line.
type LineError struct {
	Source string // file name or other description, for messages
	Line   int
	Text   string
	Reason string
}

func (e LineError) Error() string {
	return fmt.Sprintf("%s, line %d: %s (%q)", e.Source, e.Line, e.Reason, e.Text)
}

// ErrEmpty reports a list with no URLs in it.
var ErrEmpty = errors.New("the list contains no URLs")

// Read parses a list from r. source names it in error messages. Every bad line
// is reported, not only the first, so a user can fix them all at once; the
// returned error then joins one LineError per bad line.
func Read(r io.Reader, source string) (List, error) {
	var list List
	var problems []error
	seen := map[string]bool{}
	scanner := bufio.NewScanner(r)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		if reason := checkURL(text); reason != "" {
			problems = append(problems, LineError{source, lineNumber, text, reason})
			continue
		}
		if seen[text] {
			list.Duplicates = append(list.Duplicates, text)
			continue
		}
		seen[text] = true
		list.URLs = append(list.URLs, text)
	}
	if err := scanner.Err(); err != nil {
		return List{}, fmt.Errorf("read %s: %w", source, err)
	}
	if len(problems) > 0 {
		return List{}, errors.Join(problems...)
	}
	if len(list.URLs) == 0 {
		return List{}, fmt.Errorf("%s: %w", source, ErrEmpty)
	}
	return list, nil
}

// checkURL returns why text is not a crawlable URL, or "" if it is.
func checkURL(text string) string {
	parsed, err := url.Parse(text)
	if err != nil {
		return "not a valid URL"
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "must start with http:// or https://"
	}
	if parsed.Hostname() == "" {
		return "has no host name"
	}
	if strings.ContainsAny(text, " \t") {
		return "contains spaces (encode them as %20)"
	}
	return ""
}

// mustParse parses a URL that Read has already validated.
func mustParse(raw string) *url.URL {
	parsed, err := url.Parse(raw)
	if err != nil {
		panic("urllist: URL validated by Read no longer parses: " + raw)
	}
	return parsed
}
