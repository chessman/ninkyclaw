package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"ninkyclaw/pkg/model"
)

func TestConcertRows(t *testing.T) {
	rows := concertRows(model.Concert{
		// Derived from titleWidth so it keeps spilling if the column is widened.
		Title:           strings.Repeat("spill ", titleWidth),
		Date:            time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		RawTime:         "19:00",
		Source:          "concert",
		Rating:          "High",
		MatchedKeywords: []string{"bach"},
		TicketPrice:     "Paid",
		ReadMoreURL:     "https://concert.ee/a/very/long/url/that/must/not/be/wrapped",
	})

	if len(rows) < 2 {
		t.Fatalf("expected the title to wrap onto continuation rows, got %d row(s)", len(rows))
	}
	for i, row := range rows {
		if len(row) != len(headers) {
			t.Errorf("row %d has %d cells, want %d", i, len(row), len(headers))
		}
	}
	// Displays as "Link", with the full URL carried in the OSC 8 escape.
	if want := "\x1b]8;;https://concert.ee/a/very/long/url/that/must/not/be/wrapped\x1b\\Link\x1b]8;;\x1b\\"; rows[0][7] != want {
		t.Errorf("read more cell = %q, want %q", rows[0][7], want)
	}
	if got := osc8("", "Link"); got != "" {
		t.Errorf("empty URL should render nothing, got %q", got)
	}
	// Single-line fields must not repeat on continuation rows.
	if !slices.Equal(rows[1], []string{"", "", "", "", "", rows[1][5], "", ""}) {
		t.Errorf("continuation row carries more than a title: %q", rows[1])
	}
	if rows[1][5] == "" {
		t.Error("continuation row has no title text")
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		width int
		want  []string
	}{
		{"empty", "", 10, nil},
		{"fits", "short title", 20, []string{"short title"}},
		{"wraps on spaces", "one two three four", 9, []string{"one two", "three", "four"}},
		{"hard-splits a long word", "aaaaaaaaaa", 4, []string{"aaaa", "aaaa", "aa"}},
		{"flushes pending line before a long word", "hi aaaaaa", 4, []string{"hi", "aaaa", "aa"}},
		{"counts runes not bytes", "õöä ühe", 3, []string{"õöä", "ühe"}},
		{"collapses whitespace", "  a \n b  ", 10, []string{"a b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrap(tt.in, tt.width)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("wrap(%q, %d) = %q, want %q", tt.in, tt.width, got, tt.want)
			}
			for _, line := range got {
				if utf8.RuneCountInString(line) > tt.width {
					t.Errorf("line %q exceeds width %d", line, tt.width)
				}
			}
		})
	}
}

func TestWriteHTML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "concerts.html")
	if err := writeHTML(path, []model.Concert{{
		Title:           "Bach & <script>alert(1)</script>",
		Date:            time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		RawTime:         "19:00",
		Source:          "concert",
		Rating:          "High",
		MatchedKeywords: []string{"bach", "organ"},
		TicketPrice:     "Paid",
		ReadMoreURL:     "https://concert.ee/event/1",
	}}); err != nil {
		t.Fatalf("writeHTML: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)

	for _, want := range []string{
		`<a href="https://concert.ee/event/1">`,
		"Bach &amp; ",
		"2026-10-01",
		"bach, organ",
		"High",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("page is missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "<script>") {
		t.Errorf("title was not escaped:\n%s", got)
	}
}

func TestPastRowsAreMarked(t *testing.T) {
	now := time.Now()
	if isPast(now) {
		t.Error("today should not count as past")
	}
	if isPast(now.AddDate(0, 0, 1)) {
		t.Error("tomorrow should not count as past")
	}
	if !isPast(now.AddDate(0, 0, -1)) {
		t.Error("yesterday should count as past")
	}

	path := filepath.Join(t.TempDir(), "past.html")
	if err := writeHTML(path, []model.Concert{
		{Title: "gone", Date: now.AddDate(0, 0, -1)},
		{Title: "coming", Date: now.AddDate(0, 0, 1)},
	}); err != nil {
		t.Fatalf("writeHTML: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(b), `class="past"`); got != 1 {
		t.Errorf("got %d past rows, want 1:\n%s", got, b)
	}
}

func TestEventDates(t *testing.T) {
	day := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)

	for _, tt := range []struct {
		rawTime string
		want    string
	}{
		{"19:00", "20261005T190000/20261005T210000"},
		{"19:00 - 22:00", "20261005T190000/20261005T220000"},
		{"18:00 – 19:30", "20261005T180000/20261005T193000"},
		{"15:51 - 25.10.2026 23:51", "20261005T155100/20261005T235100"},
		{"", "20261005/20261006"},
		{"kell", "20261005/20261006"},
	} {
		if got := eventDates(model.Concert{Date: day, RawTime: tt.rawTime}); got != tt.want {
			t.Errorf("eventDates(%q) = %q, want %q", tt.rawTime, got, tt.want)
		}
	}
}
