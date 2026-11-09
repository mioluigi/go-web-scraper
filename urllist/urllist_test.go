package urllist

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestReadSkipsCommentsAndBlankLinesAndRemovesDuplicates(t *testing.T) {
	input := "# robots\n\nhttps://en.wikipedia.org/wiki/Robot\n  https://en.wikipedia.org/wiki/Chatbot  \n" +
		"https://en.wikipedia.org/wiki/Robot\nhttp://example.org/page"
	list, err := Read(strings.NewReader(input), "urls.txt")
	if err != nil {
		t.Fatal(err)
	}
	wantURLs := []string{"https://en.wikipedia.org/wiki/Robot", "https://en.wikipedia.org/wiki/Chatbot", "http://example.org/page"}
	if !reflect.DeepEqual(list.URLs, wantURLs) {
		t.Errorf("URLs = %v, want %v", list.URLs, wantURLs)
	}
	if !reflect.DeepEqual(list.Duplicates, []string{"https://en.wikipedia.org/wiki/Robot"}) {
		t.Errorf("Duplicates = %v", list.Duplicates)
	}
	if !reflect.DeepEqual(list.Hosts(), []string{"en.wikipedia.org", "example.org"}) {
		t.Errorf("Hosts = %v", list.Hosts())
	}
}

func TestReadReportsEveryBadLineWithItsNumber(t *testing.T) {
	input := "https://en.wikipedia.org/wiki/Robot\nftp://example.org/file\nen.wikipedia.org/wiki/Robot\nhttps://\nhttps://x.org/a b\nhttp://%zz"
	_, err := Read(strings.NewReader(input), "my-list.txt")
	if err == nil {
		t.Fatal("want an error")
	}
	for _, want := range []string{
		"my-list.txt, line 2: must start with http:// or https://",
		"line 3: must start with http:// or https://",
		"line 4: has no host name",
		"line 5: contains spaces",
		"line 6: not a valid URL",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q:\n%v", want, err)
		}
	}
	var lineErr LineError
	if !errors.As(err, &lineErr) || lineErr.Line != 2 {
		t.Errorf("first LineError = %+v, want line 2", lineErr)
	}
}

func TestReadRejectsAListWithoutURLs(t *testing.T) {
	_, err := Read(strings.NewReader("# nothing here\n\n"), "empty.txt")
	if !errors.Is(err, ErrEmpty) {
		t.Errorf("err = %v, want ErrEmpty", err)
	}
}
