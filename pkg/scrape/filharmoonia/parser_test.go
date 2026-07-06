package filharmoonia

import (
	"hash/fnv"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCalendar(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "..", "testdata", "filharmoonia_fixture.html")
	file, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("Failed to open fixture file: %v", err)
	}
	defer file.Close()

	scraper := NewScraper()
	concerts, err := scraper.Parse(file)
	if err != nil {
		t.Fatalf("Failed to parse calendar: %v", err)
	}

	if len(concerts) != 1 {
		t.Fatalf("Expected 1 concert, got %d", len(concerts))
	}

	concert := concerts[0]

	h := fnv.New32a()
	h.Write([]byte("2fe34935-6c28-468c-89bb-47740985e36a"))
	expectedID := int(h.Sum32())
	if concert.ID != expectedID {
		t.Errorf("Expected ID %d, got %d", expectedID, concert.ID)
	}

	expectedTitle := "Mozart-Beethoven-Schubert. Klaverimuusika neljale käele"
	if concert.Title != expectedTitle {
		t.Errorf("Expected Title %q, got %q", expectedTitle, concert.Title)
	}

	expectedDate := time.Date(2026, 8, 1, 13, 0, 0, 0, time.UTC)
	if !concert.Date.Equal(expectedDate) {
		t.Errorf("Expected Date %v, got %v", expectedDate, concert.Date)
	}

	expectedTime := "16:00"
	if concert.RawTime != expectedTime {
		t.Errorf("Expected RawTime %q, got %q", expectedTime, concert.RawTime)
	}

	expectedVenue := "Valge saal"
	if concert.Venue != expectedVenue {
		t.Errorf("Expected Venue %q, got %q", expectedVenue, concert.Venue)
	}

	expectedReadMoreURL := "https://www.filharmoonia.ee/event-details/mozart-beethoven-schubert"
	if concert.ReadMoreURL != expectedReadMoreURL {
		t.Errorf("Expected ReadMoreURL %q, got %q", expectedReadMoreURL, concert.ReadMoreURL)
	}

	expectedImageURL := "https://static.wixstatic.com/media/594a18.jpg"
	if concert.ImageURL != expectedImageURL {
		t.Errorf("Expected ImageURL %q, got %q", expectedImageURL, concert.ImageURL)
	}

	expectedTicketPrice := "Sold Out"
	if concert.TicketPrice != expectedTicketPrice {
		t.Errorf("Expected TicketPrice %q, got %q", expectedTicketPrice, concert.TicketPrice)
	}

	expectedSource := "filharmoonia"
	if concert.Source != expectedSource {
		t.Errorf("Expected Source %q, got %q", expectedSource, concert.Source)
	}
}
