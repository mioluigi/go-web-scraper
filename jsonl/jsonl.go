// Package jsonl writes JSON Lines (https://jsonlines.org): one JSON value per
// line, UTF-8, each line ended by "\n". It is the input format of many
// databases and knowledge-base loaders.
package jsonl

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Writer writes one JSON value per line to an underlying writer.
// It is not safe for concurrent use; give each output a single owner.
type Writer struct {
	encoder *json.Encoder
}

// NewWriter returns a Writer on w. Text is written as is: "<", ">" and "&"
// are not turned into < escapes, so scraped text stays readable.
func NewWriter(w io.Writer) *Writer {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return &Writer{encoder: encoder}
}

// Write appends value as one line. encoding/json never emits a raw newline
// inside a value (newlines in strings become \n), so one value is one line.
func (w *Writer) Write(value any) error {
	return w.encoder.Encode(value)
}

// WriteFile writes records to path as JSON Lines, atomically: it writes a
// temporary file in the same directory and renames it over path only when
// every record has been written and flushed. A failure leaves any previous
// file at path untouched and removes the temporary file.
func WriteFile[T any](path string, records []T) (err error) {
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		if err != nil {
			temporary.Close()
			os.Remove(temporary.Name())
		}
	}()
	writer := NewWriter(temporary)
	for i, record := range records {
		if err = writer.Write(record); err != nil {
			return fmt.Errorf("write record %d: %w", i+1, err)
		}
	}
	if err = temporary.Sync(); err != nil {
		return fmt.Errorf("flush output file: %w", err)
	}
	if err = temporary.Close(); err != nil {
		return fmt.Errorf("close output file: %w", err)
	}
	if err = os.Chmod(temporary.Name(), 0o644); err != nil {
		return fmt.Errorf("set output file permissions: %w", err)
	}
	if err = os.Rename(temporary.Name(), path); err != nil {
		return fmt.Errorf("move output file into place: %w", err)
	}
	return nil
}
