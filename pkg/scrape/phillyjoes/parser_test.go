package phillyjoes

import (
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseCalendar(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "..", "testdata", "phillyjoes_calendar_fixture.json")
	file, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("Failed to open calendar fixture: %v", err)
	}
	defer file.Close()

	scraper := NewScraper()
	concerts, err := scraper.Parse(file)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(concerts) == 0 {
		t.Fatalf("Expected concerts, got 0")
	}

	// Verify the first event
	first := concerts[0]
	if !strings.Contains(first.Title, "Latin Jam Session") {
		t.Errorf("Expected Title to contain 'Latin Jam Session', got %q", first.Title)
	}

	// API returns events in reverse chronological order;
	// first event is the Latin Jam Session on 2026-09-29 at 20:00 Tallinn (17:00 UTC)
	expectedDate := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	if !first.Date.Equal(expectedDate) {
		t.Errorf("Expected Date %v, got %v", expectedDate, first.Date)
	}

	if first.RawTime != "20:00" {
		t.Errorf("Expected RawTime '20:00', got %q", first.RawTime)
	}

	if first.Venue != "Philly Joe's jazz club" {
		t.Errorf("Expected Venue 'Philly Joe's jazz club', got %q", first.Venue)
	}

	expectedReadMoreURL := "https://www.phillyjoes.com/programme/latin-jam-session-29-09-26"
	if first.ReadMoreURL != expectedReadMoreURL {
		t.Errorf("Expected ReadMoreURL %q, got %q", expectedReadMoreURL, first.ReadMoreURL)
	}

	h := fnv.New32a()
	h.Write([]byte(expectedReadMoreURL))
	expectedID := int(h.Sum32())
	if first.ID != expectedID {
		t.Errorf("Expected ID %d, got %d", expectedID, first.ID)
	}

	if first.Source != "phillyjoes" {
		t.Errorf("Expected Source 'phillyjoes', got %q", first.Source)
	}
}

func TestParseDetail(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "..", "testdata", "phillyjoes_detail_fixture.html")
	file, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("Failed to open detail fixture: %v", err)
	}
	defer file.Close()

	desc, ticketPrice, ticketURL, err := ParseDetail(file)
	if err != nil {
		t.Fatalf("ParseDetail error: %v", err)
	}

	if !strings.Contains(desc, "Exactitudes, vol. 2") {
		t.Errorf("Expected description to contain 'Exactitudes, vol. 2'")
	}

	if ticketPrice != "Paid" {
		t.Errorf("Expected ticketPrice 'Paid', got %q", ticketPrice)
	}

	expectedTicketURL := "https://piletikeskus.ee/et/e/w1lnru"
	if ticketURL != expectedTicketURL {
		t.Errorf("Expected ticketURL %q, got %q", expectedTicketURL, ticketURL)
	}
}
