# SFEDD — Week 7 working notes (proposed v2.1)

> Candidate lessons and process changes found **during** Week 7. Luis confirms each one; confirmed entries
> move to the project's `SFEDD/lessons-learned.md` and `SFEDD/SFEDD-process.md` at the end of the week.
> Status: **P** proposed · **C** confirmed · **R** rejected.

## Candidate lessons so far

| ID | Category | What happened (evidence) | Lesson | Proposed action | Status |
|---|---|---|---|---|---|
| L-W7-01 | Hand-off | The builder's workspace could not reach Wikipedia (proxy 403), could not fetch `golang.org/x` modules, and had no valid GitHub token. Found in the first ten minutes (spike 001 §1). | Know what each machine can reach **before** planning; it decides where every task runs. | Phase 0 gets an **environment probe**: target sites, module proxy, GitHub read/push, toolchain versions on both machines; tasks are tagged Claude / Mac from the start. | P |
| L-W7-02 | Technical | Colly's defaults ignore robots.txt, send a library User-Agent and allow unlimited parallel requests (spike 001 §2). | A dependency's **defaults are design decisions someone else made**; read and list them in the spike. | Spike template gets a **"defaults audit"** table for every new dependency. | P |
| L-W7-03 | Data | Tests for a program that reads the live web cannot depend on the live web; a hand-built page could not settle the extraction rules (spike 001 §3). | **Record once, replay forever**: capture real inputs with provenance (URL, revision, hash, date, licence), then test offline. | Data gate (§5.2) extended to **fixtures from third parties**: manifest + licence note + hash check. | P |
| L-W7-04 | Process | The program acts on someone else's system (Wikimedia). Its rules (User-Agent policy, robots.txt, load) were missing from the rubric but are real requirements. | A third party the program touches is a **stakeholder**; its policies become NFRs. | Constitution template gets an optional **Article 9 — Respect for external systems**. | P |
| L-W7-05 | Process | The assignment's premise ("Scrapy runs sequentially") is probably wrong: Scrapy is asynchronous by default (spike P1, to be tested). | A comparison is only fair when the baseline runs **as designed and with equal settings**; test the premise, don't inherit it. | Measurement protocol (§5.3) gets a **fair-comparison rule**: equal concurrency, delays, cache; state the baseline's defaults. | P (pending T13) |
| L-W7-06 | Hand-off | Module checksums differ when the workspace resolves through mirrors. | The official machine generates `go.sum`. | Hand-off protocol: lock files are produced on the official machine. | P |
| L-W7-08 | Hand-off | A git bundle attached in chat never reached the Mac (two attempts, `no matches found`); the instruction assumed the download had happened. Fixed by attaching the repository to Claude's session with push access. | A file in the chat is not a file on disk. Prefer a channel both sides can verify (GitHub); if a file must travel, downloading it is its own step with a check. | Hand-off protocol §5.1: **no file transfer by attachment**; set up push access at intake (ties to L-W7-01). | P |
| L-W7-09 | Testing | The goroutine-leak test called `CloseClientConnections` on the test server, doing the clean-up the code was supposed to do; the mutation that deleted `CloseIdleConnections` survived (`docs/mutation-checks.txt`). | A test's own clean-up can hide the defect it exists to catch. Mutation checks find this; reading the test does not. | §5.5: leak and resource tests must not release the resource themselves. | P |
| L-W7-10 | AI review (self) | Spike 001 §3 listed "words glued together" as a Scrapy defect. On reflection that is a goquery `.Text()` artefact: Scrapy's `//text()` returns a list of fragments. The `AllText` extractor joins fragments with spaces to stay faithful. | A spike finding is a claim about *which tool* behaved so; attribute it precisely. | Spike addendum records the correction (Principle 3). | P |
| L-W7-11 | Technical | Our User-Agent followed the policy page as we read it and was refused (403); a 7-row probe on the Mac found the accepted form in two minutes (spike addendum A). | A site's policy page is a claim; **one live request is the evidence**. Probe the external system with the exact client settings before building on them. | Spike template: a live **contract probe** with the real client settings, run on the machine that can reach the site, before the spec is approved. | P |
| L-W7-07 | Design | Owner's goal is reuse "many times" (intake). | Write reuse as **requirements with tests** (FR-R1–R3), not as an aspiration; the site-specific part must be swappable. | Spec template gets a **Reuse requirements** block. | P |

## How the method changed this week (for the README's process section)

1. Intake now includes an environment probe (L-W7-01).
2. The spike audits library defaults and checks reachability, and is split into a **workspace part** and a
   **Mac part** with an explicit hand-off.
3. Fixtures from a third party carry provenance and licence, and are checked by the data gate.
4. The constitution gained an article for the external system the program touches.
5. Predictions about the baseline are written down before measuring, to be confirmed or corrected.
