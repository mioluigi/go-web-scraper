# Tasks 001 — ordered checklist

| Field | Value |
|---|---|
| Status | **APPROVED v1 — 2026-11-08** |
| Rule | One task = one commit, message starts with the ID. 🔴 = test written first and seen failing. **Mac** = Luis runs it. |

| ID | Task | Verify | Where |
|---|---|---|---|
| ✅ T00 | Repository created; intake, constitution, spike, spec, plan, tasks, capture script committed | Claude clones it and sees the files | Mac → GitHub |
| ✅ T01 | Spike Part B: `scripts/capture-pages.sh`; manifest, robots and default-UA probe committed; `go mod init` + `go get colly@v2.3.0` → `go.mod`/`go.sum` | MANIFEST has 10 × HTTP 200; Claude pulls and checks | **Mac** |
| ✅ T02 | Spike addendum: extraction selectors re-checked on the ten real pages; Scrapy info recorded | addendum committed | Claude |
| ✅ T03 | `scripts/check-inputs.sh` (fixture sizes, manifest hashes match, no stray files) — tested on pass and fail | both cases demonstrated | Claude |
| ✅ T04 | 🔴 `jsonl`: one object per line, no HTML escaping, atomic file write | tests + mutation | Claude |
| ✅ T05 | 🔴 `extract`: `WikipediaArticle` and `AllText` on the ten saved pages (AC-3) | tests + mutation | Claude |
| ✅ T06 | 🔴 `urllist`: parsing and validation with line-numbered errors (FR-2) | table tests | Claude |
| ✅ T07 | `internal/testsite`: serves testdata with latency, status codes, robots.txt | used by T08 | Claude |
| ✅ T08 | 🔴 `crawler`: Colly config, input-order results, errors per URL, robots, UA (AC-2, AC-6, AC-7) | `go test -race`, 50× leak check | Claude |
| ✅ T09 | `main.go`: flags, embedded list, summary, exit codes 0/1/2 | exit codes on the built binary | Claude |
| ✅ T10 | Benchmarks (offline, parallelism 1/2/4/8) + profiles | `go test -bench` | Claude |
| ✅ T11 | `cmd/measure` + `scripts/measure.sh` (freshness guard, machine.txt) — guard tested | pass/fail cases | Claude |
| ✅ T12 | Official run: `results/wikipedia.jl` + timing on the Mac | 10 lines; tables generated | **Mac** |
| ⏭️ T13 | Scrapy baseline run (if Q8a) | results/scrapy-*.tsv | **Mac** | *Skipped (Q8b): Scrapy page not available.*
| ✅ T14 | `scripts/tables.sh` + headline; spike addendum on P1–P3 | numbers match results | Claude |
| ✅ T15 | Builds: macOS binary, `.exe`, `.app` | `.app` runs to completion | **Mac** | *`.app` ran to completion on the Mac: "Saved 10 of 10 pages" (2026-11-08 21:00).*
| ✅ T16 | README (generated tables, licence credit, GenAI Tools) | prose re-check against tables | Claude |
| ✅ T17 | Gemini repository review + triage | every finding decided | Luis + Claude | *Two attempts; 100/100; findings decided in docs/reviews/002-repository-review.md.*
| ✅ T18 | `scripts/acceptance.sh` on a fresh clone; dialogs + index; ACs ticked | all pass | **Mac** | *Acceptance 25/25 on a fresh clone at `3aad1b7` (after the review fix and the dialog files); dialogs + index committed.*
