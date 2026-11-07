# Spec 001 — Concurrent web scraper in Go (Colly), first target: Wikipedia

| Field | Value |
|---|---|
| Status | **DRAFT v1 — 2026-11-07** · awaiting Luis's answers to Q1–Q9 and Gemini spec review |
| Inputs | `docs/assignment-and-rubric.md` · `docs/constitution.md` · `docs/spikes/001-colly-wikipedia.md` |
| Deadline | 2026-11-08 (plan and tasks drafted in parallel with review; nothing built before approval) |

## 1. Problem

The firm's knowledge base on intelligent systems and robotics needs the **text** of web pages, starting with
ten Wikipedia articles. Its Python/Scrapy scraper is considered slow. We are asked for a Go replacement built
on Colly that fetches pages concurrently, removes the HTML, and writes **one JSON object per page** to a JSON
Lines (`.jl`) file — the input format for the knowledge base. The firm expects to **reuse** the tool many times,
on other sites and larger lists.

## 2. Users and use

| User | Needs |
|---|---|
| Data scientist running a collection | Run with no arguments for the ten pages, or pass a URL list; see progress and a plain summary; get a clean `.jl` file. |
| Knowledge-base loader (a program) | One valid JSON object per line, UTF-8, the same fields every time, readable text without markup. |
| Developer reusing the tool | Call the crawler from Go code with their own URL list and their own "what to keep" rule for another site. |
| Site operator (Wikimedia) | Identified, robots-respecting, rate-bounded traffic (Constitution Art. 9). |
| Grader | Build, run and test from the README; evidence for every claim. |

## 3. Functional requirements

| ID | Requirement |
|---|---|
| **FR-1 Input** | With no arguments, crawl the ten assignment URLs (from `urls.txt`, embedded in the program). With `-urls FILE`, read one URL per line, ignoring blank lines and `#` comments. |
| **FR-2 Validation** | Reject, before any request, a list that is empty, a line that is not an absolute `http(s)` URL, or a host outside the allowed domains; name the file, line number and reason. Duplicates are crawled once and reported. |
| **FR-3 Fetching** | Fetch with Colly, concurrently, with a per-domain parallelism limit (`-parallel`, default **Q6**) and optional delay (`-delay`), a request timeout, a descriptive User-Agent (Art. 9.1), and robots.txt obeyed (Art. 9.2). |
| **FR-4 Extraction** | For each page produce `url`, `title` and `text` as defined in **Q3**, plus the metadata fields in **Q2**. The rule for what to keep is a **replaceable component** (FR-R2). |
| **FR-5 Output** | Write a JSON Lines file (`-out`, default `results/wikipedia.jl`): one object per successful page, `\n` line endings, UTF-8, no HTML-escaping of `<`, `>`, `&` in text. Order per **Q4**. Written atomically (temporary file, then rename) so a crash never leaves half a file. |
| **FR-6 Failures** | A page that fails (network, HTTP status, robots, parse) does not stop the others. Each failure is reported with URL and reason per **Q5**. |
| **FR-7 Summary and exit codes** | At the end, print to standard error: pages succeeded/failed, failures with reasons, elapsed time, output path. Exit **0** all succeeded · **1** some pages failed · **2** usage or input error (nothing fetched). |
| **FR-8 Comparison mode** | A way to time sequential (`-parallel 1`) and concurrent runs of the same list with the cache off, for the README comparison (S7). |
| **FR-9 Cache (development)** | Optional `-cache DIR` reuses saved responses, to avoid re-fetching during development (Art. 9.4). Never used for timing. |

### Reuse requirements

| ID | Requirement |
|---|---|
| **FR-R1** | Crawling is a Go package with no printing and no `os.Exit`: give it URLs, settings and an extractor; receive one result per URL (page or error). |
| **FR-R2** | An **extractor** turns one HTML document into a record. Wikipedia's rules are one extractor; another site needs only a new extractor. |
| **FR-R3** | The JSON Lines writer is its own small package, usable by any program. |

## 4. Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-1 Correctness under concurrency** | Concurrent and sequential crawls of the same saved pages give identical records (ignoring order and timestamps) — Art. 1.1. Race detector clean. |
| **NFR-2 No leaks** | Goroutine count returns to its starting value after every crawl, including crawls with failures. |
| **NFR-3 Offline tests** | Tests use saved real pages (`testdata/wikipedia/`, spike 001 Part B) served by `httptest`; no test touches the internet. |
| **NFR-4 Performance evidence** | (a) Offline benchmark: saved pages with simulated latency, parallelism 1, 2, 4, 8 — pages/s and allocations. (b) Live on the Mac: sequential vs concurrent Go, median of 5 runs each, cache off. (c) Scrapy under the same settings, if Q8 = yes. CPU and memory profiles kept. |
| **NFR-5 Politeness** | Art. 9; default parallelism and delay stated in the README. Live measurement runs total ≤ 150 requests. |
| **NFR-6 Platforms** | macOS binary, Windows `.exe`, macOS `.app` (Art. 6.1). |
| **NFR-7 Licence** | Wikipedia text in fixtures and output is credited, CC BY-SA 4.0 (Art. 8.3). |

## 5. Acceptance criteria

| ID | Criterion | Checked by |
|---|---|---|
| AC-1 | `go vet`, `gofmt -l` clean; `go test -race ./...` passes; coverage ≥ 85 % on each package | acceptance.sh |
| AC-2 | Concurrent records = sequential records on the ten saved pages | test |
| AC-3 | On every saved page: title equals the article title; text non-empty; contains none of: CSS (`{` … `}` rules), `[edit]`, citation markers `[n]`; known glued-word pairs absent | test |
| AC-4 | Built binary, no arguments, on the Mac: `results/wikipedia.jl` has **10 lines**, each valid JSON with the Q2 fields, non-empty `title` and `text` | acceptance.sh + committed file |
| AC-5 | Bad input (missing file, bad URL, empty list) → message naming the problem, exit 2; one unreachable URL → other pages written, exit 1 | acceptance.sh on the built binary |
| AC-6 | robots.txt disallowing a path → that URL reported as refused, not fetched; User-Agent header matches Art. 9.1 | test |
| AC-7 | Goroutine count restored after crawls with and without failures | test |
| AC-8 | README tables and headline numbers generated from `results/` by script; prose re-checked | tables.sh, review |
| AC-9 | `scripts/acceptance.sh` passes on a fresh clone | Luis's Mac |
| AC-10 | GenAI Tools section; dialogs in `docs/ai-dialogs/` with index | review |
| AC-11 | `.exe`, macOS binary and `.app` build; the `.app` runs to completion | Luis's Mac |

## 6. Traceability

### 6a. Assignment steps → requirements

| Step | Covered by |
|---|---|
| S1 Go + Colly | FR-3, FR-R1 |
| S2 ten Wikipedia URLs | FR-1, AC-4 |
| S3 strip markup, text only | FR-4, Q3, AC-3 |
| S4 results similar to Scrapy | Q2, Q3, Q8 (field-by-field comparison table in README) |
| S5 concurrency | FR-3, NFR-1, NFR-2, AC-2, AC-7 |
| S6 `.jl` file committed | FR-5, AC-4 |
| S7 time comparison (optional) | FR-8, NFR-4, Q8 |
| S8 README | AC-8, AC-9 |
| S9 GenAI Tools + logs | AC-10 |
| S10 `.git` URL | Phase 10 |

### 6b. Rubric → requirements

| Criterion | Covered by |
|---|---|
| [C] | Constitution Art. 2; FR-R1–R3 keep functions small and single-purpose |
| [T] | NFR-1–4, AC-1–3, AC-5–7; mutation checks; profiles |
| [D] | FR-R1–R3, FR-5 (atomic write), FR-6, Art. 9 |
| [Doc] | AC-8–10 |
| [A] | FR-2, FR-7, AC-4, AC-5, AC-11 |

## 7. Out of scope (this submission)

- Following links beyond the given list (multi-level focused crawling) — see Q7.
- JavaScript-rendered sites, logins, images, PDFs.
- Storing results in a database; the `.jl` file is the hand-off to the knowledge base.

## 8. Open decisions (Luis)

| Q | Decision | Options | Recommendation |
|---|---|---|---|
| **Q1** | Repository name | `go-web-scraper` · `go-wikipedia-scraper` · `go-colly-crawler` | **`go-web-scraper`** — the tool is meant for reuse beyond Wikipedia. Its URL is also the User-Agent contact (Art. 8.2). |
| **Q2** | Fields per record | (a) `url`, `title`, `text` only · (b) (a) + `retrieved_at`, `revision_id`, `word_count` · (c) (b) + whatever extra fields the Scrapy item has (e.g. tags) | **(b), plus (c) once we see the Scrapy code.** Revision ID and time make each record traceable to an exact article version — useful in a knowledge base. |
| **Q3** | What "text" means | (a) Scrapy-faithful: all text under the content area (spike: includes CSS, `[edit]`, `[1]`, glued words) · (b) clean article prose: paragraphs, list items and section headings, one per line, without references, navigation boxes, infoboxes, edit links · (c) both | **(b) as `text`, and (a) available as a second extractor** (`-extract scrapy`) used only to show the comparison (S4). Demonstrates FR-R2 at no extra cost. |
| **Q4** | Order of lines in the `.jl` | (a) as pages finish (streaming, constant memory) · (b) the order of the input list | **(b) for this submission** — the committed file diffs cleanly between runs; buffering ten records is free. The writer accepts either, so a large crawl can stream. |
| **Q5** | Where failures go | (a) stderr summary only · (b) also an `errors.jl` next to the output · (c) error records inside the main `.jl` | **(a) + (b)**: keep the knowledge-base file clean, keep a machine-readable failure list. |
| **Q6** | Default parallelism and delay | 2 · **4** · 8 per domain; delay 0 or 100 ms | **4, no delay.** Ten pages, an identified client; measurements also use 1, 2, 8. |
| **Q7** | Link following | (a) list only · (b) `-depth N` within allowed domains | **(a)**; the package design leaves room for (b) later. |
| **Q8** | Scrapy comparison | (a) run it unchanged on the Mac, same settings · (b) skip, explain | **(a) if** you can share the Canvas page and installing Scrapy takes under 30 min (`python3 -m venv` + `pip install scrapy`); otherwise (b). |
| **Q9** | Commit real Wikipedia pages as fixtures | (a) yes, with manifest and licence note · (b) synthetic only | **(a)** — the spike showed hand-built HTML cannot settle the extraction rules. |
