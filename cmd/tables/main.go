// Command tables turns the measured results in results/ into the Markdown
// tables and headline numbers quoted by the README (results/tables.md), so no
// number in the README is ever typed by hand (Constitution 1.3).
//
// Usage: go run ./cmd/tables [-dir results]       writes <dir>/tables.md
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	dir := flag.String("dir", "results", "folder with the measured results")
	flag.Parse()
	report, err := buildReport(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tables:", err)
		os.Exit(1)
	}
	output := filepath.Join(*dir, "tables.md")
	if err := os.WriteFile(output, []byte(report), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "tables:", err)
		os.Exit(1)
	}
	fmt.Println("wrote", output)
}

// liveRun is one line of timing.tsv, written by scripts/measure.sh.
type liveRun struct {
	parallel  int
	programMS float64 // crawl and write, measured inside the program
	processS  float64 // whole process, including start-up, from /usr/bin/time
	maxRSSMB  float64
}

// liveSummary is the median of the runs at one parallelism.
type liveSummary struct {
	parallel                     int
	runs                         int
	programS, processS, maxRSSMB float64
}

func buildReport(dir string) (string, error) {
	machine, err := os.ReadFile(filepath.Join(dir, "machine.txt"))
	if err != nil {
		return "", fmt.Errorf("machine.txt: %w (run scripts/measure.sh on the Mac first)", err)
	}
	runs, err := readLiveRuns(filepath.Join(dir, "timing.tsv"))
	if err != nil {
		return "", err
	}
	live := summarize(runs)
	if len(live) < 2 {
		return "", fmt.Errorf("timing.tsv needs runs at two parallelism settings, found %d", len(live))
	}
	benchmarks, err := readBenchmarks(filepath.Join(dir, "benchmarks.txt"))
	if err != nil {
		return "", err
	}
	clean, err := readPages(filepath.Join(dir, "wikipedia.jl"))
	if err != nil {
		return "", err
	}
	raw, err := readPages(filepath.Join(dir, "wikipedia-alltext.jl"))
	if err != nil {
		return "", err
	}
	scrapy, _ := readScrapy(filepath.Join(dir, "scrapy-timing.tsv")) // optional (spec Q8)

	var out strings.Builder
	writeHeadline(&out, live, clean, raw, scrapy)
	writeOfflineHeadline(&out, benchmarks)
	out.WriteString("\n**Table 1 — Machine**\n\n```\n" + strings.TrimSpace(string(machine)) + "\n```\n")
	writeLiveTable(&out, live)
	writeScrapyTable(&out, live, scrapy)
	writeBenchmarkTable(&out, benchmarks)
	writePagesTable(&out, clean, raw)
	return out.String(), nil
}

func readLiveRuns(path string) ([]liveRun, error) {
	rows, err := readTSV(path)
	if err != nil {
		return nil, err
	}
	var runs []liveRun
	for _, row := range rows {
		if len(row) < 5 {
			return nil, fmt.Errorf("%s: short row %v", path, row)
		}
		runs = append(runs, liveRun{atoi(row[1]), atof(row[2]), atof(row[3]), atof(row[4]) / 1e6})
	}
	return runs, nil
}

func summarize(runs []liveRun) []liveSummary {
	byParallel := map[int][]liveRun{}
	for _, run := range runs {
		byParallel[run.parallel] = append(byParallel[run.parallel], run)
	}
	var summaries []liveSummary
	for parallel, group := range byParallel {
		summaries = append(summaries, liveSummary{
			parallel: parallel,
			runs:     len(group),
			programS: median(group, func(r liveRun) float64 { return r.programMS / 1000 }),
			processS: median(group, func(r liveRun) float64 { return r.processS }),
			maxRSSMB: median(group, func(r liveRun) float64 { return r.maxRSSMB }),
		})
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].parallel < summaries[j].parallel })
	return summaries
}

func median[T any](items []T, value func(T) float64) float64 {
	values := make([]float64, len(items))
	for i, item := range items {
		values[i] = value(item)
	}
	sort.Float64s(values)
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return values[middle]
	}
	return (values[middle-1] + values[middle]) / 2
}

// benchmark is one parallelism setting of BenchmarkCrawlSavedPages.
type benchmark struct {
	name                       string
	pagesPerSecond, msPerCrawl float64
	bytesPerCrawl, allocs      float64
}

// readBenchmarks parses `go test -bench` output lines such as
// BenchmarkCrawlSavedPages/parallel=4-10  3  301234567 ns/op  33.2 pages/s  4567 B/op  89 allocs/op
func readBenchmarks(path string) ([]benchmark, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var results []benchmark
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 || !strings.HasPrefix(fields[0], "Benchmark") {
			continue
		}
		result := benchmark{name: strings.TrimPrefix(fields[0], "Benchmark")}
		for i := 2; i+1 < len(fields); i += 2 {
			value := atof(fields[i])
			switch fields[i+1] {
			case "ns/op":
				result.msPerCrawl = value / 1e6
			case "pages/s":
				result.pagesPerSecond = value
			case "B/op":
				result.bytesPerCrawl = value
			case "allocs/op":
				result.allocs = value
			}
		}
		results = append(results, result)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("%s: no benchmark lines", path)
	}
	return results, scanner.Err()
}

// pageSummary is what the tables need from one .jl record.
type pageSummary struct {
	Title     string `json:"title"`
	Text      string `json:"text"`
	WordCount int    `json:"word_count"`
}

func readPages(path string) ([]pageSummary, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var pages []pageSummary
	scanner := bufio.NewScanner(file)
	scanner.Buffer(nil, 16<<20)
	for scanner.Scan() {
		var page pageSummary
		if err := json.Unmarshal(scanner.Bytes(), &page); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, len(pages)+1, err)
		}
		pages = append(pages, page)
	}
	return pages, scanner.Err()
}

// scrapyRun is one line of the optional scrapy-timing.tsv: run, settings, process seconds.
type scrapyRun struct {
	settings string
	processS float64
}

func readScrapy(path string) (map[string]float64, error) {
	rows, err := readTSV(path)
	if err != nil {
		return nil, err
	}
	bySettings := map[string][]scrapyRun{}
	for _, row := range rows {
		if len(row) >= 3 {
			bySettings[row[1]] = append(bySettings[row[1]], scrapyRun{row[1], atof(row[2])})
		}
	}
	medians := map[string]float64{}
	for settings, group := range bySettings {
		medians[settings] = median(group, func(r scrapyRun) float64 { return r.processS })
	}
	return medians, nil
}

func writeHeadline(out *strings.Builder, live []liveSummary, clean, raw []pageSummary, scrapy map[string]float64) {
	sequential, fastest := live[0], live[len(live)-1]
	out.WriteString("**Headline (generated)**\n\n")
	fmt.Fprintf(out, "- Live crawl of the %d pages on the Mac, median of %d runs: **%.2f s** one page at a time, "+
		"**%.2f s** with %d at a time — **×%.1f** faster (whole process: %.2f s → %.2f s).\n",
		len(clean), sequential.runs, sequential.programS, fastest.programS, fastest.parallel,
		sequential.programS/fastest.programS, sequential.processS, fastest.processS)
	fmt.Fprintf(out, "- Peak memory of the whole program: %.0f MB (one at a time) and %.0f MB (%d at a time).\n",
		sequential.maxRSSMB, fastest.maxRSSMB, fastest.parallel)
	cleanChars, rawChars := totalChars(clean), totalChars(raw)
	fmt.Fprintf(out, "- The clean extraction keeps **%.0f %%** of the characters of the Scrapy-style extraction "+
		"(%s vs %s characters over all pages); the rest is styles, citation markers, edit links, "+
		"navigation boxes, tables and reference lists.\n", 100*float64(cleanChars)/float64(rawChars),
		thousands(cleanChars), thousands(rawChars))
	if original, ok := scrapy["default"]; ok {
		fmt.Fprintf(out, "- The original Scrapy program, unchanged, median **%.2f s** (whole process) versus **%.2f s** for "+
			"Go with %d at a time.\n", original, fastest.processS, fastest.parallel)
	}
}

// writeOfflineHeadline compares each parallel crawl benchmark with the one-at-a-time one.
func writeOfflineHeadline(out *strings.Builder, benchmarks []benchmark) {
	var baseline float64
	var speedups []string
	for _, b := range benchmarks {
		parallel := parallelOf(b.name)
		if parallel == 0 {
			continue
		}
		if parallel == 1 {
			baseline = b.pagesPerSecond
			continue
		}
		if baseline > 0 {
			speedups = append(speedups, fmt.Sprintf("×%.1f with %d at a time", b.pagesPerSecond/baseline, parallel))
		}
	}
	if len(speedups) > 0 {
		fmt.Fprintf(out, "- Offline, where every page waits 100 ms (simulated network): %s, versus one at a time.\n",
			strings.Join(speedups, ", "))
	}
}

// parallelOf reads N from a benchmark name such as "CrawlSavedPages/parallel=N-10"; 0 if absent.
func parallelOf(name string) int {
	_, after, found := strings.Cut(name, "parallel=")
	if !found {
		return 0
	}
	digits, _, _ := strings.Cut(after, "-")
	return atoi(digits)
}

func writeLiveTable(out *strings.Builder, live []liveSummary) {
	out.WriteString("\n**Table 2 — Live crawl of the ten pages (Mac, cache off, medians)**\n\n")
	out.WriteString("| Requests at once | Runs | Crawl + write (s) | Whole process (s) | Peak memory (MB) | Speed-up vs 1 |\n")
	out.WriteString("|---:|---:|---:|---:|---:|---:|\n")
	for _, s := range live {
		fmt.Fprintf(out, "| %d | %d | %.2f | %.2f | %.0f | ×%.1f |\n",
			s.parallel, s.runs, s.programS, s.processS, s.maxRSSMB, live[0].programS/s.programS)
	}
}

func writeScrapyTable(out *strings.Builder, live []liveSummary, scrapy map[string]float64) {
	if len(scrapy) == 0 {
		out.WriteString("\n*No Scrapy measurements (spec Q8b): see the README for why.*\n")
		return
	}
	out.WriteString("\n**Table 2b — Scrapy, whole process (Mac, medians)**\n\n| Settings | Whole process (s) |\n|---|---:|\n")
	var names []string
	for name := range scrapy {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(out, "| Scrapy, %s | %.2f |\n", name, scrapy[name])
	}
	for _, s := range live {
		fmt.Fprintf(out, "| Go, %d at a time | %.2f |\n", s.parallel, s.processS)
	}
}

func writeBenchmarkTable(out *strings.Builder, benchmarks []benchmark) {
	out.WriteString("\n**Table 3 — Offline benchmarks (`go test -bench`): crawling the saved pages with 100 ms simulated latency per page, and extraction alone**\n\n")
	out.WriteString("| Benchmark | Pages/s | ms per crawl | KB allocated per crawl | Allocations per crawl |\n")
	out.WriteString("|---|---:|---:|---:|---:|\n")
	for _, b := range benchmarks {
		fmt.Fprintf(out, "| %s | %.1f | %.0f | %s | %s |\n", b.name, b.pagesPerSecond, b.msPerCrawl,
			thousands(int(b.bytesPerCrawl/1024)), thousands(int(b.allocs)))
	}
}

func writePagesTable(out *strings.Builder, clean, raw []pageSummary) {
	out.WriteString("\n**Table 4 — What was kept, per page (`results/wikipedia.jl` vs `results/wikipedia-alltext.jl`)**\n\n")
	out.WriteString("| Page | Words (clean) | Characters (clean) | Characters (Scrapy-style) | Kept |\n|---|---:|---:|---:|---:|\n")
	rawByTitle := map[string]int{}
	for _, page := range raw {
		rawByTitle[page.Title] = len(page.Text)
	}
	for _, page := range clean {
		rawChars := rawByTitle[page.Title]
		kept := "—"
		if rawChars > 0 {
			kept = fmt.Sprintf("%.0f %%", 100*float64(len(page.Text))/float64(rawChars))
		}
		fmt.Fprintf(out, "| %s | %s | %s | %s | %s |\n", page.Title, thousands(page.WordCount),
			thousands(len(page.Text)), thousands(rawChars), kept)
	}
}

func totalChars(pages []pageSummary) int {
	total := 0
	for _, page := range pages {
		total += len(page.Text)
	}
	return total
}

// readTSV returns the rows of a tab-separated file after its header line.
func readTSV(path string) ([][]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var rows [][]string
	for _, line := range lines[1:] {
		rows = append(rows, strings.Split(line, "\t"))
	}
	return rows, nil
}

func atoi(s string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(s))
	return value
}

func atof(s string) float64 {
	value, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return value
}

// thousands formats 12345 as "12,345".
func thousands(n int) string {
	digits := strconv.Itoa(n)
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "," + digits[i:]
	}
	return digits
}
