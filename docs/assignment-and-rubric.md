# Assignment Requirements and Rubric (summary)

> Summary of the Week 7 assignment, *Crawling and Scraping the Web* (100 points, due **2026-11-08**), in our
> own words, so every tool and reviewer can check the spec against it. The Canvas page is the authoritative
> source.

## Scenario

A technology firm is building an **online library (knowledge base)** about its R&D in **intelligent systems
and robotics**. Part of the content comes from the web: an initial **crawl**, then **scraping** of each page,
then **text parsing**. The firm wrote a crawler/scraper in **Python with Scrapy** (after Mitchell 2018) and
finds it slow, believing it searches pages **sequentially**. Managers want it rewritten in **Go**, expecting
goroutines to crawl, scrape and parse many pages **concurrently** across multicore processors.

## Source to replicate

- The firm's Scrapy program: `WebFocusedCrawl/spiders/articles-spider.py`, described on the Canvas page
  *Python/Scrapy Wikipedia Example* (course login required — **not yet in our hands**, see spike 001 §5).
- Framework to use in Go: **Colly** (`github.com/gocolly/colly`), with its GitHub examples.

## Target URLs (given)

```
https://en.wikipedia.org/wiki/Robotics
https://en.wikipedia.org/wiki/Robot
https://en.wikipedia.org/wiki/Reinforcement_learning
https://en.wikipedia.org/wiki/Robot_Operating_System
https://en.wikipedia.org/wiki/Intelligent_agent
https://en.wikipedia.org/wiki/Software_agent
https://en.wikipedia.org/wiki/Robotic_process_automation
https://en.wikipedia.org/wiki/Chatbot
https://en.wikipedia.org/wiki/Applications_of_artificial_intelligence
https://en.wikipedia.org/wiki/Android_(robot)
```

The assignment suggests a Go `[]string` in place of the Python list.

## Required steps (numbered for traceability)

| # | Step | Source wording, condensed |
|---|---|---|
| S1 | Build a **Go** web crawler/scraper using **Colly** | "develop a Go-based web crawler/scraper … using the Colly framework" |
| S2 | Crawl the **10 Wikipedia URLs** above | "retrieve information (web pages) about intelligent systems and robotics from Wikipedia" |
| S3 | **Strip the HTML markup**; keep the **text**, ignore images | "scrape away the HTML markup codes … focus on the text" |
| S4 | Get **results similar to the Python/Scrapy program** | "results similar to what would be provided by a Python/Scrapy-based program" |
| S5 | Use **concurrency** across pages | management's rationale: goroutines across many pages |
| S6 | Write a **JSON Lines file** (`.jl`), **one JSON object per page** | key deliverable; committed in the repository |
| S7 | *(Optional)* **Compare processing time** with the Scrapy program; observations in the README | "not an explicit requirement … consider comparing" |
| S8 | Complete **README.md**, with run and test instructions | deliverables |
| S9 | **GenAI Tools** section + conversation logs | deliverables |
| S10 | Submit the cloneable **`.git` URL** | deliverables |

## Deliverables

- Cloneable GitHub URL ending in `.git`, in the Canvas comments.
- `README.md` as complete documentation (roles of programs and data, how to run and test).
- A **`.jl`** JSON Lines file with the text from the crawl, committed.
- **GenAI Tools** section and conversation logs (plain text or links).

## Rubric — five criteria, 20 points each

Ratings per criterion: Outstanding 20 (all criteria) · Meets 15 (most) · Needs Improvement 10 · Partially
Acceptable 5 · Unacceptable 0.

| Criterion | What earns full marks |
|---|---|
| **[C] Coding rules, organization, aesthetics** | Go modules, idiomatic Go; meaningful names; no useless temporaries; clean blocks; one task per block; simple control flow; distribute the NOT; self-documenting code (McConnell 2004, 777–817). |
| **[T] Testing and software metrics** | Unit tests of critical components with synthetic data; logs and profiles; memory and processing measured for components and the whole program; Monte Carlo benchmark *only if required* (it is not this week). |
| **[D] Design and development** | Clean, efficient design meeting every requirement; **general-purpose modules reusable in other projects**. |
| **[Doc] Documentation** | Effective Git/GitHub use; README with roles of programs and data, how to test and use. |
| **[A] Application** | An executable (`.exe` for Windows or `.app` for macOS); runs to completion; validates user input and explains mistakes; clean UX/UI. |

## Owner's emphasis (Luis, 2026-11-07)

This tool will be **reused many times** for business data collection. Design for reuse beyond Wikipedia and
beyond this assignment, and keep the method reproducible so the tool can be rebuilt and improved.
