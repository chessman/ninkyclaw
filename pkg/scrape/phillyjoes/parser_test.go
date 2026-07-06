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
	fixturePath := filepath.Join("..", "..", "..", "scratch_pj_july_2026.json")
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
	if !strings.Contains(first.Title, "Karl Martin Kirm") {
		t.Errorf("Expected Title to contain 'Karl Martin Kirm', got %q", first.Title)
	}

	// API returns events in reverse chronological order;
	// first event is Karl Martin Kirm on 2026-07-31 at 21:00 Tallinn (18:00 UTC)
	expectedDate := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	if !first.Date.Equal(expectedDate) {
		t.Errorf("Expected Date %v, got %v", expectedDate, first.Date)
	}

	if first.RawTime != "21:00" {
		t.Errorf("Expected RawTime '21:00', got %q", first.RawTime)
	}

	if first.Venue != "Philly Joe's jazz club" {
		t.Errorf("Expected Venue 'Philly Joe's jazz club', got %q", first.Venue)
	}

	expectedReadMoreURL := "https://www.phillyjoes.com/programme/resident-karl-martin-kirm-exactitudes-vol-2"
	if first.ReadMoreURL != expectedReadMoreURL {
		t.Errorf("Expected ReadMoreURL %q, got %q", expectedReadMoreURL, first.ReadMoreURL)
	}

	h := fnv.New32a()
	h.Write([]byte("6a452d8289bc560170d4d9f8"))
	expectedID := int(h.Sum32())
	if first.ID != expectedID {
		t.Errorf("Expected ID %d, got %d", expectedID, first.ID)
	}

	if first.Source != "phillyjoes" {
		t.Errorf("Expected Source 'phillyjoes', got %q", first.Source)
	}
}

func TestParseDetail(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "..", "scratch_pj_detail.html")
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
