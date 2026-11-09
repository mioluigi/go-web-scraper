package main

import (
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	good := `{"url":"u1","title":"T1","text":"x"}` + "\n" + `{"url":"u2","title":"T2","text":"y"}` + "\n"
	cases := []struct {
		name, input string
		want        int
		problem     string // "" = valid
	}{
		{"valid", good, 2, ""},
		{"valid, any count", good, -1, ""},
		{"wrong count", good, 3, "2 lines, want 3"},
		{"not JSON", good + "{oops\n", -1, "line 3 is not a JSON object"},
		{"empty text", `{"url":"u","title":"T","text":""}`, -1, `line 1 has no "text"`},
		{"missing title", `{"url":"u","text":"x"}`, -1, `line 1 has no "title"`},
		{"empty file", "", -1, "file is empty"},
	}
	for _, c := range cases {
		_, err := check(strings.NewReader(c.input), c.want)
		if c.problem == "" && err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
		}
		if c.problem != "" && (err == nil || !strings.Contains(err.Error(), c.problem)) {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.problem)
		}
	}
}
