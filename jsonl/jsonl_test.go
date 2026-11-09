package jsonl

import (
	"bufio"
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type record struct {
	Name string `json:"name"`
	Text string `json:"text"`
}

func TestWriterPutsOneValuePerLineAndKeepsMarkupCharacters(t *testing.T) {
	var buffer bytes.Buffer
	writer := NewWriter(&buffer)
	input := []record{
		{"first", "Robots & agents <AI>"},
		{"second", "line one\nline two"}, // an embedded newline must not split the record
	}
	for _, r := range input {
		if err := writer.Write(r); err != nil {
			t.Fatal(err)
		}
	}
	want := `{"name":"first","text":"Robots & agents <AI>"}` + "\n" +
		`{"name":"second","text":"line one\nline two"}` + "\n"
	if buffer.String() != want {
		t.Errorf("got\n%s\nwant\n%s", buffer.String(), want)
	}
}

func TestWriteFileRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pages.jl")
	input := []record{{"a", "alpha"}, {"b", "beta"}, {"c", "Ünïcödé — ok"}}
	if err := WriteFile(path, input); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var got []record
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var r record
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			t.Fatalf("line %d is not JSON: %v", len(got)+1, err)
		}
		got = append(got, r)
	}
	if len(got) != len(input) {
		t.Fatalf("%d lines, want %d", len(got), len(input))
	}
	for i := range input {
		if got[i] != input[i] {
			t.Errorf("line %d = %+v, want %+v", i+1, got[i], input[i])
		}
	}
}

func TestWriteFileFailureKeepsThePreviousFileAndLeavesNoTemporary(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "pages.jl")
	if err := os.WriteFile(path, []byte("previous\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := WriteFile(path, []float64{1, math.NaN()}) // NaN cannot be encoded as JSON
	if err == nil || !strings.Contains(err.Error(), "record 2") {
		t.Fatalf("err = %v, want an error naming record 2", err)
	}
	if content, _ := os.ReadFile(path); string(content) != "previous\n" {
		t.Errorf("previous file changed to %q", content)
	}
	if entries, _ := os.ReadDir(directory); len(entries) != 1 {
		t.Errorf("directory has %d entries, want only the previous file", len(entries))
	}
}

func TestWriteFileReportsAMissingDirectory(t *testing.T) {
	err := WriteFile(filepath.Join(t.TempDir(), "missing", "pages.jl"), []record{{"a", "b"}})
	if err == nil || !strings.Contains(err.Error(), "create output file") {
		t.Errorf("err = %v, want a create error", err)
	}
}
