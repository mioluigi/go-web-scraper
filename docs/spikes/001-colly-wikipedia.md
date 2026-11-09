# Spike 001 — Can we build and test a Colly scraper here, what does "the text" mean, and what will concurrency buy?

| Field | Value |
|---|---|
| Date | 2026-11-07 |
| Machine | Claude's Linux workspace, 2 CPUs, Go 1.24.7 (indicative only; official numbers come from Luis's Mac, Go 1.27.2) |
| Code | Colly **v2.3.0** (latest tag), goquery; harness `001-evidence/spike_main.go.txt` |
| Input | `001-evidence/wikipedia-shaped-fixture.html` — hand-built to mirror Wikipedia's page structure (not a real page; see Part B) |
| Raw output | `001-evidence/spike-output.txt` |

## Questions

1. Can Claude's workspace reach Wikipedia and the Go module servers? (decides where each kind of work happens)
2. What do Colly's **defaults** do that matter for a polite, correct crawler? (S1, Art. 9)
3. What does a Scrapy-style "all the text" extraction produce, and what should "the text" be? (S3, S4)
4. Do Colly's callbacks run concurrently, does its limit hold, and does anything leak? (S5, Art. 2.8)
5. What will concurrency gain on ten pages, and against what baseline? (S7)

## Part A — Results in Claude's workspace

### 1. Reachability: Wikipedia is blocked here; the Go proxy is partial ⚠️

| Target | Result | Consequence |
|---|---|---|
| `en.wikipedia.org` | **403 at the egress proxy** (policy) | No live crawl from the workspace. Every live run happens on Luis's Mac; tests must use saved pages served by `httptest`. |
| `proxy.golang.org` for `golang.org/x/*` | Forbidden | Resolved here through GitHub mirrors (`replace` lines, workspace only). The hashes differ from the official modules, so **`go.sum` must be generated on the Mac** and the `replace` lines never committed. |
| `github.com` (git, read) | OK | Can clone public repos (e.g. Week 6) and verify pushes. |
| GitHub API with a token | Invalid token | Claude cannot push; Luis pushes, Claude verifies by cloning. |

*This took ten minutes and changed the plan — an argument for making it a standing intake check (proposed
L-W7-01).*

### 2. Colly v2.3.0 defaults (read in `colly.go`, `Init()`, and confirmed by running)

| Setting | Default | Why it matters |
|---|---|---|
| `IgnoreRobotsTxt` | **`true`** — robots.txt is *not* obeyed unless we turn it on | Art. 9.2 requires obeying it. |
| `UserAgent` | `colly - https://github.com/gocolly/colly` (server saw exactly this) | Wikimedia asks automated clients for a descriptive UA with contact information and may block generic library defaults without notice. Art. 9.1. |
| `Async(true)` without a `LimitRule` | No bound on parallel requests | We must set `Parallelism` per domain. |
| `MaxBodySize` | 10 MB | Wikipedia articles are well under this; keep. |
| Responses parsed | Status < 203 only; others go to `OnError` | Failures arrive through `OnError`; they need to become per-URL error reports. |
| Module `go` directive | `go 1.24.0` | Fine for Go 1.27 on the Mac; our `go.mod` must say ≥ 1.24. |

### 3. "The text": the Scrapy-style extraction is noisy ⚠️

Taking all text under `#mw-content-text` (what an XPath `//text()` over the content div does) on the fixture
gave **419 characters**, including:

- CSS from Wikipedia's inline template styles: `.mw-parser-output .hatnote{font-style:italic}`
- Edit links and citation markers: `History[edit]`, `robots.[1]`
- **Words glued together** where block elements touch: `FieldEngineering`, `engineeringComputer science`
- Navigation-box and reference-list text.

A cleaned extraction (remove `style`, `script`, `sup.reference`, `.mw-editsection`, `.navbox`, `.reflist`,
`.hatnote`, infobox; then take `p`, `li`, `h2–h4` one per line) gave **232 characters** of readable prose with
section headings.

**Implication.** "Results similar to Scrapy" (S4) and "scrape away the HTML markup" (S3) pull in different
directions if the Scrapy program keeps everything. This is open decision Q3 in the spec. *Caveat:* the
fixture is hand-built; the selectors must be confirmed on **real pages** (Part B).

### 4. Concurrency behaviour ✅

12 requests to a local server that waits 50 ms each, `Parallelism: 4`:

- **169 ms** total versus ≈ 600 ms one-at-a-time (×3.5 with 4 in flight).
- Peak concurrent requests seen by the server: **4** — the limit holds exactly.
- Peak concurrent `OnHTML` callbacks: **4** — callbacks run **in parallel**, so the output file needs a single
  writer (mutex or channel). Art. 2.8.
- Goroutines before 2, after 2 — no leak once connections are closed.

### 5. Baseline: the Scrapy program is not in our hands yet ❓

The *Python/Scrapy Wikipedia Example* page is behind the course login. Without it we cannot run the inherited
code unchanged (SFEDD phase 1), know its output fields, or time it. **Luis to provide it** (Part B, step 3).

## Predictions (hypotheses to check against the Mac's measurements)

- **P1 — The premise is half right.** Scrapy is asynchronous (Twisted) and by default runs up to 16 requests
  at once, 8 per domain, so it is probably **not** fetching sequentially. If so, Go's advantage on ten pages
  will come mostly from **start-up time** (Python and Scrapy imports) and much less from concurrency.
- **P2 — The network dominates.** Downloading takes most of the wall time; parsing ten pages in Go takes tens
  of milliseconds in total. Concurrency (in either language) is what shortens a crawl.
- **P3 — Go concurrent vs Go sequential** on the ten live pages: ×3–×6 with parallelism 4–8, capped by the
  slowest single page (*Applications of artificial intelligence* is the largest).

## Part B — Live spike on Luis's Mac (hand-off)

These steps check what the workspace cannot. They follow the hand-off protocol (SFEDD §5.1): one action per
block, manual steps separate, each with a stop condition. *They run after the repository exists* (spec
Q1); the script is in `scripts/capture-pages.sh`.

1. Capture the ten pages politely (one at a time, 1 s apart, descriptive User-Agent) into
   `testdata/wikipedia/`, with a manifest of URL, HTTP status, bytes, SHA-256, revision ID and capture time.
   Also fetch `robots.txt` once and record whether `/wiki/` is allowed.
2. Send one request with Colly's **default** User-Agent and record the status (does Wikipedia accept it?).
3. Provide the Scrapy page (save the Canvas page as PDF or paste the code); if Python + Scrapy are installed,
   run it **unchanged** and record its wall time, settings and output fields.

The spike is complete when the manifest, robots result, default-UA status and Scrapy information are
committed, and the extraction selectors from §3 are re-checked on the real pages (addendum to this file).

---

## Addendum A — 2026-11-08: Wikipedia's User-Agent rule, measured on the Mac

The first live request of Part B (`capture-pages.sh`, User-Agent `go-web-scraper-capture/0.1
(https://github.com/mioluigi/go-web-scraper) curl/8.x`) was refused: **HTTP 403**, body *"Please set a
user-agent and respect our robot policy https://w.wiki/4wJS. See also
https://phabricator.wikimedia.org/T400119."* (Wikimedia's 2025 enforcement of its User-Agent policy). Single
requests to `https://en.wikipedia.org/wiki/Robotics` from Luis's Mac, about one second apart:

| User-Agent sent | Status |
|---|---|
| `Mozilla/5.0` (browser imitation; for comparison only) | 200 |
| `go-web-scraper/1.0 (https://github.com/mioluigi/go-web-scraper)` | **403** |
| `go-web-scraper/1.0 (+https://github.com/mioluigi/go-web-scraper)` | **403** |
| `go-web-scraper/1.0 (https://github.com/mioluigi/go-web-scraper) colly/2.3.0` | **403** |
| `go-web-scraper/1.0 (https://github.com/mioluigi/go-web-scraper; <e-mail>)` | **403** |
| `CoolBot/0.0 (https://example.org/coolbot/; coolbot@example.org)` (the policy page's example) | 200 |
| `go-web-scraper/1.0 (<e-mail>)` | **200** |

**Finding.** In this test, every agent carrying our `github.com` repository URL was refused, even with an
e-mail address added. A URL on another domain plus an e-mail passed, and so did the e-mail address alone. We
do not know Wikimedia's rule, only this behaviour. We adopt the form that passed:
`go-web-scraper/1.0 (<e-mail>)`, with the address supplied at run time (Constitution 8.2 and 9.1 amended).

**Correction to §2 and to the constitution as first drafted.** We planned a repository URL as the contact so
that no e-mail address would be needed. The live site ruled that out. The spike's reading of the policy page
was right that a contact is required, but wrong about which contact the site accepts.

## Addendum B — correction to §3

§3 listed "words glued together" as a defect of the Scrapy-style extraction. That came from goquery's
`.Text()` in the spike harness, not from Scrapy: Scrapy's `//text()` returns a *list* of fragments. The
`AllText` extractor therefore joins fragments with spaces, staying faithful to Scrapy. CSS, `[edit]` and
citation markers remain genuine differences.

## Addendum C — 2026-11-08: the real pages (Part B complete)

Ten pages were captured on the Mac (`testdata/wikipedia/MANIFEST.tsv`: 10 × HTTP 200, 205 KB to 1.56 MB,
revision IDs recorded); `robots.txt` does not disallow `/wiki/`. Colly's *default* User-Agent got 200 on the
probe, but the program does not use it (Art. 9.1). Running the extraction tests on them found:

1. **The live markup is sectioned.** Each section is a `<section>` that starts with its heading. The
   hand-built fixture (and the extractor) assumed flat markup, so *See also*, *References* and
   *Further reading* were kept on every page. Seen as a citation-like `[2019]` in a *Further reading* entry.
   Fixed: the extractor handles both shapes, and a sectioned synthetic fixture with exactly the same
   expected text was added. This is the risk §3's caveat named: "the selectors must be confirmed on real pages".
2. **The glued-words check was wrong, not the text.** A "lower-case then capital" pattern flagged real names
   (RoboHon, iRobot, ImageNet, AlphaGo, MaxEnt). Replaced by a check of actual block boundaries:
   for every neighbouring pair of list items or paragraphs, the last word of one joined to the first word
   of the next must not occur in the text. Both new checks were mutation-tested.

Result on the ten pages: the clean text keeps 29–47 % of the characters of the Scrapy-style extraction
(exact figures are generated in README Table 4).

## Addendum D — 2026-11-08: predictions checked against the official run (`results/tables.md`)

- **P1 (Scrapy is already concurrent): not tested.** The Scrapy program was not available (spec Q8b), so it
  stays a hypothesis, stated as such in the README.
- **P2 (the network dominates): held for a sequential crawl.** The offline crawl benchmark's time per crawl,
  minus its simulated waiting, leaves a small CPU share per page next to the live time per page (Tables 2
  and 3).
- **P3 (×3–×6 live with 4–8 at a time): failed.** Live, 4 at a time gave the speed-up in the headline,
  well below the prediction, while the offline benchmark at 4 came close to it. Wikipedia answered quickly
  from its cache, so fixed costs weighed more: robots.txt first, connection set-up, CPU parsing, and the
  largest page. These causes are likely but were not measured separately. Lesson: a speed-up prediction must
  state the latency it assumes; ours silently assumed a slow network.
