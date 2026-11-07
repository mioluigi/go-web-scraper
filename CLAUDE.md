# CLAUDE.md — project context

Crawling and Scraping the Web (Week 7): a reusable Go scraper built on Colly that fetches the ten assignment
Wikipedia pages concurrently, removes the HTML, and writes one JSON object per page to a JSON Lines file —
the input for the firm's knowledge base. Process: SFEDD v2.0 (+ proposed v2.1 notes in `docs/sfedd-notes.md`).

## Read these first (in order)

1. `docs/constitution.md` — non-negotiable rules (rubric-derived; Article 9 = respect for the target site)
2. `specs/001-wikipedia-scraper/spec.md` — what and why
3. `specs/001-wikipedia-scraper/plan.md` — how
4. `specs/001-wikipedia-scraper/tasks.md` — work the next unchecked task
5. `docs/spikes/001-colly-wikipedia.md` — what was learned before planning

## Working rules

- One task per commit; message starts with the task ID (`T05: …`).
- Test first; mutation-check important tests (compile failures do not count).
- **Tests never touch the internet**: they use `testdata/wikipedia/` through `internal/testsite`.
- Live requests to Wikipedia happen only on Luis's Mac, with the descriptive User-Agent, and only when a
  task needs them.
- Claude's workspace cannot reach Wikipedia or `golang.org/x` modules: never commit the workspace-only
  `replace` lines; `go.sum` comes from the Mac.
- Never hand-type results; generate them from `results/`. Re-check prose when results change.
- Changes after approval go in `docs/decisions.md` (newest first).
