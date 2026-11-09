package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Synthetic results with answers worked out by hand: parallel 1 runs take
// 4, 5, 6 s (median 5); parallel 4 runs take 1, 1.2, 1.4 s (median 1.2):
// speed-up 5 / 1.2 = ×4.2. Clean text 30 of 100 characters → 30 %.
func TestReportFromSyntheticResults(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("machine.txt", "machine: test\n")
	write("timing.tsv", "run\tparallel\tprogram_ms\tprocess_s\tmax_rss_bytes\n"+
		"1\t1\t4000\t4.1\t30000000\n1\t4\t1000\t1.1\t40000000\n2\t1\t6000\t6.1\t30000000\n"+
		"2\t4\t1400\t1.5\t40000000\n3\t1\t5000\t5.1\t32000000\n3\t4\t1200\t1.3\t42000000\n")
	write("benchmarks.txt", "goos: darwin\nBenchmarkCrawlSavedPages/parallel=1-10  1  1000000000 ns/op  10.0 pages/s  2048 B/op  1500 allocs/op\nPASS\n")
	write("wikipedia.jl", `{"title":"Robot","text":"`+strings.Repeat("a", 30)+`","word_count":1}`+"\n")
	write("wikipedia-alltext.jl", `{"title":"Robot","text":"`+strings.Repeat("b", 100)+`"}`+"\n")

	report, err := buildReport(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"**5.00 s** one page at a time, **1.20 s** with 4 at a time — **×4.2** faster",
		"keeps **30 %**",
		"| 1 | 3 | 5.00 | 5.10 | 30 | ×1.0 |",
		"| 4 | 3 | 1.20 | 1.30 | 40 | ×4.2 |",
		"| CrawlSavedPages/parallel=1-10 | 10.0 | 1000 | 2 | 1,500 |",
		"| Robot | 1 | 30 | 100 | 30 % |",
		"No Scrapy measurements",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report lacks %q\n%s", want, report)
		}
	}
}

func TestReportRequiresMeasurements(t *testing.T) {
	if _, err := buildReport(t.TempDir()); err == nil || !strings.Contains(err.Error(), "measure.sh") {
		t.Errorf("err = %v, want a pointer to measure.sh", err)
	}
}

func TestThousands(t *testing.T) {
	for n, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 1234567: "1,234,567"} {
		if got := thousands(n); got != want {
			t.Errorf("thousands(%d) = %q, want %q", n, got, want)
		}
	}
}
