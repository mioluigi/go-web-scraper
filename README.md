# go-web-scraper

A reusable, concurrent web scraper in Go, built on [Colly](https://github.com/gocolly/colly). Its first job is
to fetch ten Wikipedia articles on intelligent systems and robotics, keep their **text**, and write **one JSON
object per page** to a JSON Lines file, ready to load into the firm's knowledge base.

*MSDS 431 Learning Go, Week 7: Crawling and Scraping the Web.* Repository:
`https://github.com/mioluigi/go-web-scraper.git`

**Contents:** [Quick start](#quick-start) · [What it produces](#what-it-produces) · [How it works](#how-it-works) ·
[Results](#results) · [Go vs Scrapy](#go-vs-scrapy) · [Reusing it](#reusing-it-for-another-site) ·
[Testing](#testing) · [Measuring and building](#measuring-and-building) · [Files](#roles-of-programs-and-data) ·
[How it was built](#how-it-was-built-sfedd) · [GenAI Tools](#genai-tools) · [Licence](#licence-and-credits)

## Quick start

You need Go 1.24 or newer, an internet connection, and an e-mail address. Wikipedia refuses automated
requests that don't say who sends them, so the program puts your address in its User-Agent header. It is never
written to any file.

```bash
git clone https://github.com/mioluigi/go-web-scraper.git && cd go-web-scraper
export SCRAPER_CONTACT=you@example.org   # once per terminal (or pass -contact you@example.org)
go run .                                 # the ten built-in pages → results/wikipedia.jl
```

```
Crawling 10 pages, up to 4 at a time per site…
[ 1/10] ok    Robot (… words)
…
Saved 10 of 10 pages to results/wikipedia.jl in … s.
```

Ready-made programs: `scripts/build.sh` makes `dist/go-web-scraper.exe` (Windows) and macOS binaries;
`scripts/build-macos-app.sh` makes `dist/WebScraper.app`. Double-click the app and it runs in Terminal, asks for
the contact address, and saves `~/Desktop/WebScraper-output/wikipedia.jl`.

| Option | Default | Meaning |
|---|---|---|
| `-urls FILE` | built-in list (`urls.txt`) | one URL per line; blank lines and `# comments` ignored |
| `-out FILE.jl` | `results/wikipedia.jl` | output file; its folder is created; failures go to `FILE.errors.jl` |
| `-parallel N` | `4` | most requests at once **per site**, 1–32 (`1` = one at a time) |
| `-delay D` | `0` | pause before each request to a site, e.g. `250ms` |
| `-timeout D` | `30s` | give up on a page after this long |
| `-extract NAME` | `wikipedia` | `wikipedia` = clean article prose · `alltext` = every text node, like the Scrapy program |
| `-cache DIR` | off | reuse pages saved in `DIR` (development only, never for timing) |
| `-contact EMAIL` | `$SCRAPER_CONTACT` | **required**: the address sent in the User-Agent |

The program checks its input before sending any request. A bad option, a missing file or a malformed URL
gets a message naming the problem (with the line number) and **exit status 2**. If any page fails, the others
are still saved, the failures are listed with reasons, and the **exit status is 1**. **0** means every page was
saved.

## What it produces

`results/wikipedia.jl` holds one line per page. Each line is a JSON object (shortened here):

```json
{"url":"https://en.wikipedia.org/wiki/Robotics","title":"Robotics","text":"Robotics is the interdisciplinary study …\nHistory\n…","word_count":4321,"revision_id":1254321987,"retrieved_at":"2026-11-08T23:10:04Z"}
```

| Field | Meaning |
|---|---|
| `url` | the page requested |
| `title` | the article's title |
| `text` | the article's prose: paragraphs, list items and section headings, **one per line**, with no HTML |
| `word_count` | words in `text` |
| `revision_id` | Wikipedia's revision number, so each record points to an exact version of the article |
| `retrieved_at` | when it was fetched (UTC) |

**What "the text" means.** The `wikipedia` extractor removes style sheets, citation markers (`[1]`), `[edit]`
links, infoboxes, navigation boxes, tables, images and captions, and the back matter (*See also*,
*References*, *External links* and similar sections). Formulas become their plain TeX form (`r_{t}<1`). Block
elements become line breaks, so neighbouring paragraphs never run together. The `alltext` extractor keeps
every text node in the content area, as a Scrapy `//text()` extraction does. `results/wikipedia-alltext.jl`
contains that version, and Table 4 compares the two.

## How it works

```
urls.txt / -urls ──► urllist ──► crawler (Colly) ──► extract.Extractor ──► jsonl ──► results/*.jl
                     validate     N requests at      what to keep from     one object
                     + dedupe     once per site       each HTML page        per line
```

| Package | Job | Reusable as |
|---|---|---|
| `crawler/` | Fetches a list of URLs with Colly, concurrently, with a per-site limit; returns one result per URL **in input order**. No printing, no `os.Exit`. | `crawler.Crawl(urls, config, extractor)` from any Go program |
| `extract/` | `Extractor` interface; `WikipediaArticle` and `AllText`; `BlockText`, a block-aware HTML-to-text function for any site | write one `Extractor` per new site |
| `jsonl/` | JSON Lines writer (no HTML escaping) and atomic `WriteFile` | any program that writes `.jl` |
| `urllist/` | Reads URL lists, reporting **every** bad line with its number | any URL-driven tool |
| `main.go` | Flags, progress, summary, exit codes | — |

**Concurrency.** Colly sends requests from its own goroutines. A `LimitRule` caps how many are in flight per
site; the tests confirm the cap holds exactly. Callbacks run in parallel, so every result goes through one
mutex into a slice indexed by input position: there is a single writer, and the output order is stable. A
sequential run is the same code with `-parallel 1`. The tests require it to produce **identical records** to a
concurrent run. After each crawl, idle connections are closed, so no goroutine outlives the crawl. A
mutation check showed that without this, 20 crawls left 120 goroutines behind.

**Politeness** (Constitution, Article 9). Colly by default ignores `robots.txt`, sends a generic User-Agent,
and has no request limit. This program changes all three: it **obeys robots.txt**, stays on the sites in the
list, sends at most 4 requests at a time per site by default, and identifies itself as
`go-web-scraper/1.0 (<your e-mail>)`.

That last form was **measured, not assumed**. On 2026-11-08 Wikipedia answered **403** ("Please set a user-agent
and respect our robot policy") to every User-Agent that carried this repository's `github.com` URL, with or
without an e-mail address added, and to the capture script's first attempt. It answered **200** to the e-mail
address alone, and to the example from its own policy page. The evidence is in
`docs/spikes/001-colly-wikipedia.md` (addendum A). A browser-imitating User-Agent also got 200, but
Wikimedia's policy forbids that and so do we.

## Results

Measured on Luis's Mac with `scripts/measure.sh`. **Everything between the markers is generated** from
`results/` by `go run ./cmd/tables` and `scripts/readme.sh`; nothing is typed by hand.

<!-- tables:start -->
**Headline (generated)**

- Live crawl of the 10 pages on the Mac, median of 5 runs: **0.53 s** one page at a time, **0.30 s** with 4 at a time — **×1.7** faster (whole process: 0.56 s → 0.33 s).
- Peak memory of the whole program: 41 MB (one at a time) and 48 MB (4 at a time).
- The clean extraction keeps **37 %** of the characters of the Scrapy-style extraction (276,784 vs 751,602 characters over all pages); the rest is styles, citation markers, edit links, navigation boxes, tables and reference lists.

**Table 1 — Machine**

```
machine: MacBookPro18,2 · Apple M1 Max · 64 GB · macOS 27.0.1
cpus: 10
go: go1.27.2
commit: f7cbf1e
measured: 2026-11-08 20:37 EST
```

**Table 2 — Live crawl of the ten pages (Mac, cache off, medians)**

| Requests at once | Runs | Crawl + write (s) | Whole process (s) | Peak memory (MB) | Speed-up vs 1 |
|---:|---:|---:|---:|---:|---:|
| 1 | 5 | 0.53 | 0.56 | 41 | ×1.0 |
| 4 | 5 | 0.30 | 0.33 | 48 | ×1.7 |

*No Scrapy measurements (spec Q8b): see the README for why.*

**Table 3 — Offline benchmarks (`go test -bench`): crawling the saved pages with 100 ms simulated latency per page, and extraction alone**

| Benchmark | Pages/s | ms per crawl | KB allocated per crawl | Allocations per crawl |
|---|---:|---:|---:|---:|
| CrawlSavedPages/parallel=1-10 | 9.3 | 1073 | 72,453 | 548,447 |
| CrawlSavedPages/parallel=2-10 | 18.2 | 550 | 72,524 | 548,484 |
| CrawlSavedPages/parallel=4-10 | 29.9 | 335 | 72,657 | 548,640 |
| CrawlSavedPages/parallel=8-10 | 43.9 | 228 | 72,692 | 548,866 |
| WikipediaArticle-10 | 255.8 | 39 | 19,072 | 141,328 |

**Table 4 — What was kept, per page (`results/wikipedia.jl` vs `results/wikipedia-alltext.jl`)**

| Page | Words (clean) | Characters (clean) | Characters (Scrapy-style) | Kept |
|---|---:|---:|---:|---:|
| Robotics | 4,712 | 31,260 | 85,738 | 36 % |
| Robot | 9,187 | 58,709 | 130,433 | 45 % |
| Reinforcement learning | 4,907 | 33,212 | 88,962 | 37 % |
| Robot Operating System | 3,671 | 23,876 | 59,718 | 40 % |
| Intelligent agent | 2,501 | 17,080 | 45,380 | 38 % |
| Software agent | 2,484 | 16,393 | 34,939 | 47 % |
| Robotic process automation | 2,107 | 13,920 | 36,979 | 38 % |
| Chatbot | 2,988 | 19,597 | 66,640 | 29 % |
| Applications of artificial intelligence | 6,823 | 46,937 | 156,997 | 30 % |
| Android (robot) | 2,510 | 15,800 | 45,816 | 34 % |
<!-- tables:end -->

## Go vs Scrapy

<!-- scrapy:start -->
*(Written after the measurements.)*
<!-- scrapy:end -->

## Reusing it for another site

The crawler knows nothing about Wikipedia. To scrape another site, write an `Extractor`: a type that turns one
parsed HTML document into a `Page`.

```go
type BlogPost struct{}

func (BlogPost) Extract(doc *goquery.Document) (extract.Page, error) {
	title := strings.TrimSpace(doc.Find("article h1").First().Text())
	body := doc.Find("article .post-body")
	if title == "" || body.Length() == 0 {
		return extract.Page{}, extract.ErrNotArticle
	}
	return extract.Page{Title: title, Text: extract.BlockText(body)}, nil
}

results, err := crawler.Crawl(urls, crawler.DefaultConfig(), BlogPost{})
// write the pages with jsonl.WriteFile("results/blog.jl", pages)
```

`crawler.Config` sets parallelism, delay, timeout, User-Agent, a development cache, and an `OnResult` hook for
progress. The crawler fetches only the list it is given. Following links (multi-level focused crawling) is
the natural next step; the design leaves room for it (spec §7).

## Testing

```bash
go test ./...                        # all tests; none touch the internet
go test -race ./...                  # race detector
go test -cover ./...                 # coverage per package
go test -bench . -benchmem ./crawler/ ./extract/   # offline throughput benchmarks
scripts/acceptance.sh                # every acceptance check, as run before submission
```

- **No test uses the live internet.** `scripts/capture-pages.sh` saved the ten real pages once, politely, to
  `testdata/wikipedia/`, with a manifest of source, status, size, SHA-256, revision and capture time.
  `internal/testsite` serves them over `httptest`, adding latency, error statuses or a robots.txt when a
  test needs them.
- **Extraction quality** is tested on all ten real pages: correct title, at least 300 words, a revision ID,
  and no CSS rules, citation markers, `[edit]` or run-together words. A hand-written synthetic article has an
  exactly known expected text.
- **Concurrency**: concurrent records equal sequential records; the per-site limit holds; one bad page
  (HTTP 404 or 500, not an article, a timeout, refused by robots.txt) never stops the others; the User-Agent
  is sent; no goroutine leaks across 20 crawls with failures.
- **The command line**: exit codes 0, 1 and 2, every validation message, the errors file, and the built-in
  list. `scripts/acceptance.sh` checks the exit codes again **on the built binary**.
- **Mutation checks** (`docs/mutation-checks.txt`): 12 deliberate defects, each caught by a test. One was
  first missed because the test closed the connections itself; the test was fixed.
- **The checks are tested too.** `check-inputs.sh`, `capture-pages.sh`, `cmd/checkjl` and `cmd/tables` were
  each run on a case that must pass and one that must fail.

## Measuring and building

| Script | Does |
|---|---|
| `scripts/capture-pages.sh` | saves the pages used by the tests (one request per second) |
| `scripts/check-inputs.sh` | data gate: manifest hashes, sizes, count, no stray files |
| `scripts/measure.sh` | **Mac only**; refuses a stale or modified copy. Live runs at `-parallel 1` and `4`, 5 runs each, plus the official output and offline benchmarks with CPU and memory profiles. About 120 live requests in total. |
| `go run ./cmd/tables` | `results/` → `results/tables.md` (tables and headline) |
| `scripts/readme.sh` | inserts `results/tables.md` into this README |
| `scripts/build.sh` · `scripts/build-macos-app.sh` | Windows `.exe`, macOS binaries · `WebScraper.app` |
| `scripts/acceptance.sh` | the final checks on a fresh clone |

## Roles of programs and data

| Path | Role |
|---|---|
| `main.go`, `urls.txt` | the program; the ten assignment URLs, built into it |
| `crawler/`, `extract/`, `jsonl/`, `urllist/` | reusable packages (above) |
| `internal/testsite/` | test web server over saved pages |
| `cmd/checkjl/`, `cmd/tables/` | validates a `.jl` file; generates the README tables |
| `testdata/wikipedia/` | the ten saved pages + `MANIFEST.tsv`, `robots.txt`, `PROBES.txt` (CC BY-SA, below) |
| `results/` | **deliverable** `wikipedia.jl`; `wikipedia-alltext.jl`; timings, benchmarks, profiles, machine |
| `docs/`, `specs/` | the process record: assignment, constitution, spike, spec, plan, tasks, decisions, reviews, AI dialogs |

## How it was built (SFEDD)

Built with *Spec-First, Evidence-Driven Development*. The stages were intake
(`docs/assignment-and-rubric.md`), a constitution (`docs/constitution.md`), a spike that tested the risky
assumptions before planning (`docs/spikes/001-colly-wikipedia.md`), then the spec, plan and tasks
(`specs/001-wikipedia-scraper/`). After that came test-first tasks, one commit each, and measurements on the
official machine. Every number in this README is generated. `docs/decisions.md` records each change made
after approval. `docs/sfedd-notes.md` lists what this week changed in the method itself; for example,
saving third-party pages once with full provenance so tests run offline, and checking a library's defaults
before relying on them.

## GenAI Tools

<!-- genai:start -->
*(Completed at the end.)*
<!-- genai:end -->

## Licence and credits

Text in `testdata/wikipedia/` and `results/*.jl` comes from Wikipedia articles (listed in `urls.txt`;
revisions in each record and in `testdata/wikipedia/MANIFEST.tsv`), © Wikipedia contributors, under the
[Creative Commons Attribution-ShareAlike 4.0](https://creativecommons.org/licenses/by-sa/4.0/) licence. Colly is
Apache-2.0; goquery is BSD-3-Clause.
