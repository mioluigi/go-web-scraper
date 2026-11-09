// Command checkjl checks that a file is valid JSON Lines of scraped pages:
// every line one JSON object with non-empty "url", "title" and "text", and,
// with -want N, exactly N lines. It is used by scripts/acceptance.sh.
//
// Usage: checkjl [-want N] FILE      exit 0 valid · 1 invalid · 2 usage
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	want := flag.Int("want", -1, "required number of lines (-1: any)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: checkjl [-want N] FILE")
		os.Exit(2)
	}
	file, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "checkjl:", err)
		os.Exit(1)
	}
	defer file.Close()
	lines, err := check(file, *want)
	if err != nil {
		fmt.Fprintf(os.Stderr, "checkjl: %s: %v\n", flag.Arg(0), err)
		os.Exit(1)
	}
	fmt.Printf("%s: %d valid page records\n", flag.Arg(0), lines)
}

// check validates every line and returns how many there are.
func check(r io.Reader, want int) (int, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(nil, 16<<20) // long articles make long lines
	lines := 0
	for scanner.Scan() {
		lines++
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return lines, fmt.Errorf("line %d is not a JSON object: %w", lines, err)
		}
		for _, field := range []string{"url", "title", "text"} {
			if value, _ := record[field].(string); value == "" {
				return lines, fmt.Errorf("line %d has no %q", lines, field)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return lines, err
	}
	if lines == 0 {
		return 0, errors.New("file is empty")
	}
	if want >= 0 && lines != want {
		return lines, fmt.Errorf("%d lines, want %d", lines, want)
	}
	return lines, nil
}
