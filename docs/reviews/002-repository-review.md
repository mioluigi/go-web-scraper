# Repository review 002 — Gemini (web), triage

| Field | Value |
|---|---|
| Prompt | `docs/reviews/002-repository-review-prompt.md` |
| Attempt 1 | 2026-11-08 21:14 — Gemini received **only the prompt and `results/tables.md`**; the 12 code and document files did not attach. Raw review: `docs/ai-dialogs/gemini-repository-review.pdf`. |

## Attempt 1 — what Gemini could and could not see

Gemini said so itself and declined to score the rubric, the assignment steps or the code. That is the
evidence-citing rule (SFEDD §5.4) working as intended: no score on evidence it did not have. It is also a
hand-off failure: nothing checked which attachments arrived (lesson L-W7-14).

**Inaccurate statement recorded (§5.4):** "Crawling and writing 10 pages takes 0.56 s sequentially …
versus 0.33 s concurrently". Those are the **whole-process** column of Table 2. The crawl + write column is
0.53 s and 0.30 s. The ×1.7 speed-up is the same either way.

## Findings and decisions

| # | Severity (Gemini) | Finding | Test of the claim (evidence) | Decision |
|---|---|---|---|---|
| 1 | High | Program will crash if `results/` does not exist; add `os.MkdirAll`. | `main.go` `writeOutputs` calls `os.MkdirAll(filepath.Dir(output), 0o755)` before writing. `TestRunWritesOnePagePerLineAndExitsZero` writes into a folder that does not exist (`new-folder/pages.jl`). | **Reject**: already done and tested. |
| 2 | Medium | Ensure non-zero exit codes on failure. | `main.go`: exit 1 when pages fail, exit 2 for bad input. Tested by `TestRunReportsFailedPagesAndExitsOne` and `TestRunRejectsUnusableInputWithExitTwo` (12 cases), and on the **built binary** by `acceptance.sh` (no contact, missing file, bad URL, `-parallel 0`, unreachable page). | **Reject**: already done and tested. |
| 3 | Medium | Validate one JSON object per line with `\n`. | `jsonl` tests: one value per line, an embedded newline stays inside its record, round trip. `cmd/checkjl -want 10` validates both committed `.jl` files in `acceptance.sh` (25/25 on a fresh clone). | **Reject**: already done and tested. |
| 5 | — | Grader friction: missing contact e-mail, output folder. | Contact: the program refuses to run without one and prints the fix (`TestRunRequiresAContactAddress`); README Quick start shows `export SCRAPER_CONTACT=…`. Folder: see #1. | **Accept the property**, no change needed. Quick start already covers it. |

## Attempt 2

Re-run with every file attached as `.txt` and an instruction to **list the files received before
reviewing** (prompt updated). Findings will be triaged below.
