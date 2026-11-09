# Prompt for Gemini — repository review 002 (SFEDD §7.2)

**Attach these files** (not only the URL): `README.md`, `results/tables.md`, `docs/assignment-and-rubric.md`,
`docs/constitution.md`, `specs/001-wikipedia-scraper/spec.md`, `docs/spikes/001-colly-wikipedia.md`,
`docs/decisions.md`, `crawler/crawler.go`, `extract/wikipedia.go`, `extract/text.go`, `main.go`,
`crawler/crawler_test.go`, `extract/extract_test.go`, and the first 3 lines of `results/wikipedia.jl`.
Repository: https://github.com/mioluigi/go-web-scraper.git

---

You are two reviewers at once: a **strict grader** applying the attached rubric (five criteria, 20 points
each) and a **senior Go engineer** who has run production web crawlers. Review the attached repository files
for the Week 7 assignment (a Go/Colly scraper of ten Wikipedia pages writing a JSON Lines file). The
submission is due tonight, so only changes that take under 20 minutes are actionable.

Report, in this order:

1. **Score each rubric criterion** (0/5/10/15/20) with the evidence for it.
2. **Assignment steps S1–S10** (`docs/assignment-and-rubric.md`): met, partly met or missing, with evidence.
3. **Risky code**: concurrency (`crawler.go`), extraction (`wikipedia.go`, `text.go`), error handling and exit
   codes (`main.go`). Name any bug with a concrete input that triggers it.
4. **Every README conclusion checked against `results/tables.md`**: quote both. Flag any number in the
   README that is not in the tables.
5. **Grader friction**: anything a grader cloning the repository would trip over (setup, the required
   contact e-mail, missing files, unclear instructions).
6. The three changes that would raise the score most, each under 20 minutes.

**For every claim, quote the file and section it relies on.** If you cannot see something, say so; do not
assume. Each finding: severity (High / Medium / Low), where, problem, suggested change.
