// Command go-web-scraper fetches web pages concurrently, keeps their text, and
// writes one JSON object per page to a JSON Lines file.
//
// With no arguments it crawls the ten Wikipedia articles on intelligent
// systems and robotics listed in urls.txt (built into the program) and writes
// results/wikipedia.jl. Run with -h for the options.
//
// Exit status: 0 every page succeeded · 1 some pages failed, or the output
// could not be written · 2 the command line or URL list was unusable, so
// nothing was fetched.
package main

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mioluigi/go-web-scraper/crawler"
	"github.com/mioluigi/go-web-scraper/extract"
	"github.com/mioluigi/go-web-scraper/jsonl"
	"github.com/mioluigi/go-web-scraper/urllist"
)

//go:embed urls.txt
var assignmentURLs string

// Exit statuses.
const (
	exitOK           = 0
	exitPagesFailed  = 1
	exitUnusableArgs = 2
)

// Limits on user settings: enough for real use, small enough to stay polite.
const (
	maxParallelism = 32
	defaultOutput  = "results/wikipedia.jl"
)

// contactVariable names the environment variable holding the contact e-mail.
const contactVariable = "SCRAPER_CONTACT"

// errMissingContact explains the one setting with no usable default.
var errMissingContact = errors.New("a contact e-mail address is required.\n" +
	"Wikipedia refuses automated requests that do not say who is sending them. Set it once per terminal:\n" +
	"  export " + contactVariable + "=you@example.org\n" +
	"or pass -contact you@example.org. It is sent only in the User-Agent header and never saved")

// options are the command-line settings after validation.
type options struct {
	urlsFile  string // "" means the built-in list
	output    string
	extractor extract.Extractor
	crawl     crawler.Config
}

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

// run is the whole program; it returns the exit status. Messages go to
// console (standard error), so a script can still read the .jl file cleanly.
func run(arguments []string, console io.Writer) int {
	settings, err := parseOptions(arguments, console)
	if errors.Is(err, flag.ErrHelp) {
		return exitOK
	}
	if err != nil {
		fmt.Fprintf(console, "go-web-scraper: %v\nRun go-web-scraper -h for help.\n", err)
		return exitUnusableArgs
	}
	list, err := loadURLs(settings.urlsFile)
	if err != nil {
		fmt.Fprintf(console, "go-web-scraper: the URL list cannot be used:\n%v\n", err)
		return exitUnusableArgs
	}
	for _, duplicate := range list.Duplicates {
		fmt.Fprintf(console, "note: %s is listed more than once; crawling it once\n", duplicate)
	}

	fmt.Fprintf(console, "Crawling %d pages, up to %d at a time per site…\n", len(list.URLs), settings.crawl.Parallelism)
	settings.crawl.OnResult = progressReporter(console, len(list.URLs))
	started := time.Now()
	results, err := crawler.Crawl(list.URLs, settings.crawl, settings.extractor)
	if err != nil {
		fmt.Fprintf(console, "go-web-scraper: %v\n", err)
		return exitUnusableArgs
	}
	elapsed := time.Since(started)

	if err := writeOutputs(settings.output, results); err != nil {
		fmt.Fprintf(console, "go-web-scraper: %v\n", err)
		return exitPagesFailed
	}
	return reportSummary(console, results, settings, elapsed)
}

// parseOptions reads and checks the command line.
func parseOptions(arguments []string, console io.Writer) (options, error) {
	defaults := crawler.DefaultConfig()
	flags := flag.NewFlagSet("go-web-scraper", flag.ContinueOnError)
	flags.SetOutput(console)
	urlsFile := flags.String("urls", "", "file with one URL per line (default: the ten built-in Wikipedia URLs)")
	output := flags.String("out", defaultOutput, "JSON Lines file to write (its folder is created if needed)")
	parallel := flags.Int("parallel", defaults.Parallelism, fmt.Sprintf("most requests at once per site, 1–%d (1 = one at a time)", maxParallelism))
	delay := flags.Duration("delay", 0, "pause before each request to a site, e.g. 250ms")
	timeout := flags.Duration("timeout", defaults.Timeout, "give up on a page after this long")
	extractorName := flags.String("extract", "wikipedia", "what to keep: "+strings.Join(extract.Names(), " | ")+" (alltext = every text node, like the Scrapy program)")
	cacheDir := flags.String("cache", "", "reuse pages saved in this folder (development only; never for timing)")
	contact := flags.String("contact", os.Getenv(contactVariable), "your e-mail address, sent in the User-Agent as Wikipedia requires (default: $"+contactVariable+")")
	flags.Usage = func() { printUsage(flags) }
	if err := flags.Parse(arguments); err != nil {
		return options{}, err
	}

	if flags.NArg() > 0 {
		return options{}, fmt.Errorf("unexpected argument %q (put URLs in a file and use -urls FILE)", flags.Arg(0))
	}
	if *parallel < 1 || *parallel > maxParallelism {
		return options{}, fmt.Errorf("-parallel must be between 1 and %d, not %d", maxParallelism, *parallel)
	}
	if *delay < 0 {
		return options{}, fmt.Errorf("-delay cannot be negative (%v)", *delay)
	}
	if *timeout <= 0 {
		return options{}, fmt.Errorf("-timeout must be positive, e.g. 30s (got %v)", *timeout)
	}
	extractor, ok := extract.ByName(*extractorName)
	if !ok {
		return options{}, fmt.Errorf("-extract must be one of %s, not %q", strings.Join(extract.Names(), ", "), *extractorName)
	}
	if !strings.HasSuffix(*output, ".jl") {
		return options{}, fmt.Errorf("-out should end in .jl (JSON Lines), not %q", *output)
	}

	if !crawler.ValidContact(*contact) {
		return options{}, errMissingContact
	}

	settings := options{urlsFile: *urlsFile, output: *output, extractor: extractor, crawl: defaults}
	settings.crawl.Parallelism = *parallel
	settings.crawl.Delay = *delay
	settings.crawl.Timeout = *timeout
	settings.crawl.CacheDir = *cacheDir
	settings.crawl.UserAgent = crawler.UserAgent(*contact)
	return settings, nil
}

func printUsage(flags *flag.FlagSet) {
	out := flags.Output()
	fmt.Fprintln(out, "go-web-scraper — fetch web pages concurrently and save their text as JSON Lines.")
	fmt.Fprintln(out, "\nUsage:\n  export "+contactVariable+"=you@example.org   once: Wikipedia requires a contact address")
	fmt.Fprintln(out, "  go-web-scraper                         the ten built-in Wikipedia pages → "+defaultOutput)
	fmt.Fprintln(out, "  go-web-scraper -urls my-list.txt -out results/mine.jl\n\nOptions:")
	flags.PrintDefaults()
	fmt.Fprintln(out, "\nExit status: 0 all pages saved · 1 some pages failed · 2 bad options or URL list.")
}

// loadURLs reads the URL list from a file, or uses the built-in list.
func loadURLs(path string) (urllist.List, error) {
	if path == "" {
		return urllist.Read(strings.NewReader(assignmentURLs), "built-in list")
	}
	file, err := os.Open(path)
	if err != nil {
		return urllist.List{}, fmt.Errorf("cannot open %s: %w", path, unwrapPathError(err))
	}
	defer file.Close()
	return urllist.Read(file, path)
}

// unwrapPathError drops the repeated path from *os.PathError messages.
func unwrapPathError(err error) error {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err
	}
	return err
}

// progressReporter prints one line per finished page.
func progressReporter(console io.Writer, total int) func(int, crawler.Result) {
	done := 0
	return func(_ int, result crawler.Result) { // the crawler never overlaps these calls
		done++
		if result.OK() {
			fmt.Fprintf(console, "[%2d/%d] ok    %s (%d words)\n", done, total, result.Page.Title, result.Page.WordCount)
			return
		}
		fmt.Fprintf(console, "[%2d/%d] FAIL  %s — %v\n", done, total, result.Page.URL, result.Err)
	}
}

// failureRecord is one line of the errors file.
type failureRecord struct {
	URL    string `json:"url"`
	Status int    `json:"status,omitempty"`
	Error  string `json:"error"`
}

// writeOutputs writes the pages file and, if any page failed, the errors file
// beside it; a stale errors file from an earlier run is removed.
func writeOutputs(output string, results []crawler.Result) error {
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("cannot create the output folder: %w", err)
	}
	var pages []extract.Page
	var failures []failureRecord
	for _, result := range results {
		if result.OK() {
			pages = append(pages, result.Page)
		} else {
			failures = append(failures, failureRecord{result.Page.URL, result.Status, result.Err.Error()})
		}
	}
	if err := jsonl.WriteFile(output, pages); err != nil {
		return err
	}
	errorsFile := errorsPath(output)
	if len(failures) == 0 {
		if err := os.Remove(errorsFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("cannot remove the old %s: %w", errorsFile, err)
		}
		return nil
	}
	return jsonl.WriteFile(errorsFile, failures)
}

// errorsPath names the errors file: results/wikipedia.jl → results/wikipedia.errors.jl.
func errorsPath(output string) string {
	return strings.TrimSuffix(output, ".jl") + ".errors.jl"
}

// reportSummary prints the outcome and returns the exit status.
func reportSummary(console io.Writer, results []crawler.Result, settings options, elapsed time.Duration) int {
	succeeded := 0
	for _, result := range results {
		if result.OK() {
			succeeded++
		}
	}
	failed := len(results) - succeeded
	fmt.Fprintf(console, "\nSaved %d of %d pages to %s in %.2f s.\n", succeeded, len(results), settings.output, elapsed.Seconds())
	if failed > 0 {
		fmt.Fprintf(console, "%d page(s) failed; details in %s.\n", failed, errorsPath(settings.output))
	}
	// One machine-readable line for scripts/measure.sh.
	fmt.Fprintf(console, "summary pages=%d ok=%d failed=%d parallel=%d elapsed_ms=%d\n",
		len(results), succeeded, failed, settings.crawl.Parallelism, elapsed.Milliseconds())
	if failed > 0 {
		return exitPagesFailed
	}
	return exitOK
}
