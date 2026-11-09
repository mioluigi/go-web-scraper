package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mioluigi/go-web-scraper/extract"
	"github.com/mioluigi/go-web-scraper/internal/testsite"
	"github.com/mioluigi/go-web-scraper/urllist"
)

// siteWithArticles serves the synthetic article as /wiki/A and /wiki/B.
func siteWithArticles(t *testing.T) *testsite.Site {
	t.Helper()
	article, err := os.ReadFile("extract/testdata/article.html")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, name := range []string{"A", "B"} {
		os.WriteFile(filepath.Join(dir, name+".html"), article, 0o644)
	}
	site := testsite.Start(testsite.Options{Dir: dir})
	t.Cleanup(site.Close)
	return site
}

func writeList(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "urls.txt")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readPages(t *testing.T, path string) []extract.Page {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var pages []extract.Page
	scanner := bufio.NewScanner(file)
	scanner.Buffer(nil, 10<<20)
	for scanner.Scan() {
		var page extract.Page
		if err := json.Unmarshal(scanner.Bytes(), &page); err != nil {
			t.Fatalf("line %d is not a page: %v", len(pages)+1, err)
		}
		pages = append(pages, page)
	}
	return pages
}

func TestRunWritesOnePagePerLineAndExitsZero(t *testing.T) {
	site := siteWithArticles(t)
	output := filepath.Join(t.TempDir(), "new-folder", "pages.jl")
	var console bytes.Buffer
	status := run([]string{"-urls", writeList(t, site.URL("/wiki/A"), site.URL("/wiki/B")), "-out", output}, &console)
	if status != exitOK {
		t.Fatalf("status %d, console:\n%s", status, console.String())
	}
	pages := readPages(t, output)
	if len(pages) != 2 || pages[0].URL != site.URL("/wiki/A") || pages[1].Title != "Robotics" {
		t.Errorf("pages = %+v", pages)
	}
	for _, want := range []string{"Crawling 2 pages", "ok    Robotics", "Saved 2 of 2 pages", "summary pages=2 ok=2 failed=0 parallel=4"} {
		if !strings.Contains(console.String(), want) {
			t.Errorf("console lacks %q:\n%s", want, console.String())
		}
	}
	if _, err := os.Stat(errorsPath(output)); !os.IsNotExist(err) {
		t.Error("an errors file was written although nothing failed")
	}
}

func TestRunReportsFailedPagesAndExitsOne(t *testing.T) {
	site := siteWithArticles(t)
	output := filepath.Join(t.TempDir(), "pages.jl")
	var console bytes.Buffer
	status := run([]string{"-urls", writeList(t, site.URL("/wiki/A"), site.URL("/wiki/Missing")), "-out", output}, &console)
	if status != exitPagesFailed {
		t.Fatalf("status %d, want %d", status, exitPagesFailed)
	}
	if pages := readPages(t, output); len(pages) != 1 {
		t.Errorf("%d pages written, want 1", len(pages))
	}
	failures, _ := os.ReadFile(errorsPath(output))
	if !strings.Contains(string(failures), `"error":"HTTP 404 Not Found"`) {
		t.Errorf("errors file = %s", failures)
	}
	if !strings.Contains(console.String(), "1 page(s) failed") {
		t.Errorf("console:\n%s", console.String())
	}

	// A later clean run removes the stale errors file.
	run([]string{"-urls", writeList(t, site.URL("/wiki/A")), "-out", output}, &console)
	if _, err := os.Stat(errorsPath(output)); !os.IsNotExist(err) {
		t.Error("stale errors file was kept")
	}
}

func TestRunRejectsUnusableInputWithExitTwo(t *testing.T) {
	cases := map[string]struct {
		arguments []string
		message   string
	}{
		"missing list":     {[]string{"-urls", "no-such-file.txt"}, "cannot open no-such-file.txt: no such file or directory"},
		"bad URL":          {[]string{"-urls", writeList(t, "www.example.org/page")}, "line 1: must start with http:// or https://"},
		"empty list":       {[]string{"-urls", writeList(t, "# nothing")}, urllist.ErrEmpty.Error()},
		"parallel zero":    {[]string{"-parallel", "0"}, "-parallel must be between 1 and 32"},
		"parallel huge":    {[]string{"-parallel", "500"}, "-parallel must be between 1 and 32"},
		"negative delay":   {[]string{"-delay", "-1s"}, "-delay cannot be negative"},
		"zero timeout":     {[]string{"-timeout", "0s"}, "-timeout must be positive"},
		"unknown extract":  {[]string{"-extract", "scrapy"}, "-extract must be one of wikipedia, alltext"},
		"wrong extension":  {[]string{"-out", "pages.json"}, "-out should end in .jl"},
		"stray argument":   {[]string{"https://en.wikipedia.org/wiki/Robot"}, "unexpected argument"},
		"unknown flag":     {[]string{"-fast"}, "flag provided but not defined: -fast"},
		"malformed number": {[]string{"-parallel", "four"}, "invalid value"},
	}
	for name, c := range cases {
		var console bytes.Buffer
		if status := run(c.arguments, &console); status != exitUnusableArgs {
			t.Errorf("%s: status %d, want %d", name, status, exitUnusableArgs)
		}
		if !strings.Contains(console.String(), c.message) {
			t.Errorf("%s: console lacks %q:\n%s", name, c.message, console.String())
		}
	}
}

func TestHelpExitsZero(t *testing.T) {
	var console bytes.Buffer
	if status := run([]string{"-h"}, &console); status != exitOK {
		t.Errorf("status %d", status)
	}
	if !strings.Contains(console.String(), "Exit status: 0 all pages saved") {
		t.Errorf("help text:\n%s", console.String())
	}
}

func TestBuiltInListIsTheTenAssignmentURLs(t *testing.T) {
	list, err := loadURLs("")
	if err != nil {
		t.Fatal(err)
	}
	if len(list.URLs) != 10 || list.URLs[0] != "https://en.wikipedia.org/wiki/Robotics" ||
		list.URLs[9] != "https://en.wikipedia.org/wiki/Android_(robot)" {
		t.Errorf("built-in list = %v", list.URLs)
	}
}
