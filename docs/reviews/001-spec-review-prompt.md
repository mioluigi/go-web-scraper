# Prompt for Gemini — spec review 001 (SFEDD §7.1)

**Attach these files** (not only the repository URL): `docs/assignment-and-rubric.md`, `docs/constitution.md`,
`docs/spikes/001-colly-wikipedia.md`, `specs/001-wikipedia-scraper/spec.md`, `specs/001-wikipedia-scraper/plan.md`.

---

You are two reviewers at once: a **strict grader** applying the attached rubric, and a **senior Go engineer**
who has run production web crawlers. Review the attached draft spec (with its constitution, spike and plan)
for a Go/Colly scraper of ten Wikipedia pages that writes a JSON Lines file. The deadline is tomorrow.

Report, in this order:

1. **Coverage gaps** — any assignment step (S1–S10) or rubric criterion not covered, or covered only weakly.
2. **Contradictions** between spec, constitution, spike and plan.
3. **Untestable requirements** — anything that cannot be checked by a test, a script or a reading of a file.
4. **Domain risks** in crawling and scraping: politeness, Wikipedia's policies, robots.txt, encoding, redirects,
   timeouts, extraction quality, concurrency and resource leaks, fairness of the Scrapy comparison.
5. **Evidence check** — is each spike finding supported by its evidence? Is any prediction (P1–P3) unjustified?
6. **A view on each open decision Q1–Q9**: agree with the recommendation or argue for another option.
7. **Scope cuts** if the deadline is at risk, in the order you would cut them.

For each finding give: **severity** (High / Medium / Low), **where** (file and section — quote it), **problem**,
**suggested change**. Express worries as **properties a test can check**, not as a required mechanism.
Finish with a verdict: *approve*, *approve with changes*, or *rework*.
