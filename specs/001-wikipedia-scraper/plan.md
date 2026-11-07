# Plan 001 — How we build it

| Field | Value |
|---|---|
| Status | **DRAFT v1 — 2026-11-07** · written alongside the spec review; assumes the recommended answers to Q1–Q9 and changes if they change |
| Spec | `spec.md` (DRAFT v1) |

## 1. Shape of the code

```
go-web-scraper/
  main.go                 command line: flags, URL list, summary, exit codes (thin)
  urls.txt                the ten assignment URLs, embedded with go:embed
  crawler/                FR-R1  Crawl(urls, Config, Extractor) []Result — Colly inside, no printing
  extract/                FR-R2  Extractor interface; WikipediaArticle (clean); AllText (Scrapy-like)
  jsonl/                  FR-R3  Writer (one object per line, no HTML escaping); atomic WriteFile
  urllist/                FR-1/2 parse and validate a URL list with line-numbered errors
  internal/testsite/      httptest server over testdata/: latency, status codes, robots.txt
  testdata/wikipedia/     ten saved pages + MANIFEST.tsv + PROBES.txt (spike Part B)
  cmd/measure/            repeated timed crawls → results/*.tsv (sequential, concurrent; memory)
  scripts/                capture-pages · check-inputs · measure · tables · readme · build · acceptance
  results/                wikipedia.jl, errors.jl, timing, profiles, machine.txt (from the Mac)
```

## 2. Key design choices (each traced to a spec item)

| Choice | Why |
|---|---|
| **One Colly collector per crawl**, `Async(true)`, `LimitRule{DomainGlob: "*", Parallelism: cfg.Parallelism, Delay: cfg.Delay}` | Spike §4: the limit holds exactly. Sequential = the same code with `Parallelism: 1` (NFR-1, Art. 4.2). |
| Each request carries its **input position** in Colly's request context; results go into a pre-sized slice under one mutex | Callbacks run concurrently (spike §4); one writer; input order for free (Q4b). |
| `IgnoreRobotsTxt = false`, `AllowedDomains` from the URL list, descriptive UA set explicitly | Spike §2 defaults; Art. 9. A robots refusal comes back from `Visit` immediately and becomes that URL's error. |
| Errors from `OnError`, from `Visit`, and from the extractor all become `Result.Err` | FR-6; no goroutine waits on anything after `Wait()` returns (NFR-2). |
| Extractor works on a **cloned** document: remove noise selectors, then read `p, li, h2–h4` one per line | Spike §3; to be re-checked on real pages before T05. |
| `jsonl.Writer` uses `json.Encoder` with `SetEscapeHTML(false)`; `WriteFile` writes `*.tmp` then renames | FR-5 readable text; never half a file. |
| Revision ID read from the page's `wgRevisionId`; `retrieved_at` from the response time | Q2(b) traceability. |
| Tests serve `testdata/` from `internal/testsite`, which rewrites nothing — the crawler is pointed at the test server's host | NFR-3; AllowedDomains comes from the list, so it works for `127.0.0.1` too. |

## 3. Measurement

- **Offline** (`go test -bench`, any machine): ten saved pages, simulated latency 100 ms, parallelism 1/2/4/8;
  pages per second, allocations; CPU and memory profiles. Shows the concurrency effect without touching
  Wikipedia.
- **Live** (`scripts/measure.sh`, Mac only, freshness guard): built binary, cache off, `-parallel 1` and
  `-parallel 4`, 5 runs each, alternating; wall time and peak memory (`/usr/bin/time -l`). 100 requests.
- **Scrapy** (if Q8a): the original spider, unchanged, 5 runs, its settings recorded; then once with
  `CONCURRENT_REQUESTS_PER_DOMAIN=4` to match ours. ≤ 60 requests.
- `scripts/tables.sh` turns `results/` into `results/tables.md` and the headline block; P1–P3 checked in a
  spike addendum.

## 4. Risks

| Risk | Mitigation |
|---|---|
| Real page structure differs from the spike fixture | Spike Part B captures real pages before the extractor is written. |
| Wikipedia rate-limits or blocks during measurement | Descriptive UA, parallelism ≤ 8, ≤ 150 requests, pauses between runs; failures surface as exit 1 and stop `measure.sh`. |
| Workspace and Mac resolve modules differently | `go.sum` generated on the Mac (T01); `acceptance.sh` on a fresh clone. |
| Scrapy unavailable in time | Q8b: skip with an explanation; Go-vs-Go comparison stands on its own. |
