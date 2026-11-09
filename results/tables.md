**Headline (generated)**

- Live crawl of the 10 pages on the Mac, median of 5 runs: **0.53 s** one page at a time, **0.30 s** with 4 at a time — **×1.7** faster (whole process: 0.56 s → 0.33 s).
- Peak memory of the whole program: 41 MB (one at a time) and 48 MB (4 at a time).
- The clean extraction keeps **37 %** of the characters of the Scrapy-style extraction (276,784 vs 751,602 characters over all pages); the rest is styles, citation markers, edit links, navigation boxes, tables and reference lists.

**Table 1 — Machine**

```
machine: MacBookPro18,2 · Apple M1 Max · 64 GB · macOS 27.0.1
cpus: 10
go: go1.27.2
commit: f7cbf1e
measured: 2026-11-08 20:37 EST
```

**Table 2 — Live crawl of the ten pages (Mac, cache off, medians)**

| Requests at once | Runs | Crawl + write (s) | Whole process (s) | Peak memory (MB) | Speed-up vs 1 |
|---:|---:|---:|---:|---:|---:|
| 1 | 5 | 0.53 | 0.56 | 41 | ×1.0 |
| 4 | 5 | 0.30 | 0.33 | 48 | ×1.7 |

*No Scrapy measurements (spec Q8b): see the README for why.*

**Table 3 — Offline benchmarks (`go test -bench`): crawling the saved pages with 100 ms simulated latency per page, and extraction alone**

| Benchmark | Pages/s | ms per crawl | KB allocated per crawl | Allocations per crawl |
|---|---:|---:|---:|---:|
| CrawlSavedPages/parallel=1-10 | 9.3 | 1073 | 72,453 | 548,447 |
| CrawlSavedPages/parallel=2-10 | 18.2 | 550 | 72,524 | 548,484 |
| CrawlSavedPages/parallel=4-10 | 29.9 | 335 | 72,657 | 548,640 |
| CrawlSavedPages/parallel=8-10 | 43.9 | 228 | 72,692 | 548,866 |
| WikipediaArticle-10 | 255.8 | 39 | 19,072 | 141,328 |

**Table 4 — What was kept, per page (`results/wikipedia.jl` vs `results/wikipedia-alltext.jl`)**

| Page | Words (clean) | Characters (clean) | Characters (Scrapy-style) | Kept |
|---|---:|---:|---:|---:|
| Robotics | 4,712 | 31,260 | 85,738 | 36 % |
| Robot | 9,187 | 58,709 | 130,433 | 45 % |
| Reinforcement learning | 4,907 | 33,212 | 88,962 | 37 % |
| Robot Operating System | 3,671 | 23,876 | 59,718 | 40 % |
| Intelligent agent | 2,501 | 17,080 | 45,380 | 38 % |
| Software agent | 2,484 | 16,393 | 34,939 | 47 % |
| Robotic process automation | 2,107 | 13,920 | 36,979 | 38 % |
| Chatbot | 2,988 | 19,597 | 66,640 | 29 % |
| Applications of artificial intelligence | 6,823 | 46,937 | 156,997 | 30 % |
| Android (robot) | 2,510 | 15,800 | 45,816 | 34 % |
