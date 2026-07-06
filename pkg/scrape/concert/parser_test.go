package concert

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCalendar(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "..", "testdata", "concert_calendar_fixture.html")
	file, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("Failed to open fixture file: %v", err)
	}
	defer file.Close()

	scraper := NewScraper()
	concerts, err := scraper.Parse(file, 2026, 12)
	if err != nil {
		t.Fatalf("Failed to parse calendar: %v", err)
	}

	if len(concerts) != 1 {
		t.Fatalf("Expected 1 concert, got %d", len(concerts))
	}

	concert := concerts[0]

	expectedID := 5366
	if concert.ID != expectedID {
		t.Errorf("Expected ID %d, got %d", expectedID, concert.ID)
	}

	expectedTitle := "Haydn. Loomine"
	if concert.Title != expectedTitle {
		t.Errorf("Expected Title %q, got %q", expectedTitle, concert.Title)
	}

	expectedDate := time.Date(2026, 12, 19, 0, 0, 0, 0, time.UTC)
	if !concert.Date.Equal(expectedDate) {
		t.Errorf("Expected Date %v, got %v", expectedDate, concert.Date)
	}

	expectedTime := "19:00"
	if concert.RawTime != expectedTime {
		t.Errorf("Expected RawTime %q, got %q", expectedTime, concert.RawTime)
	}

	expectedVenue := "Estonia kontserdisaal"
	if concert.Venue != expectedVenue {
		t.Errorf("Expected Venue %q, got %q", expectedVenue, concert.Venue)
	}

	expectedReadMoreURL := "https://concert.ee/kontsert/haydn-loomine/?ek_id=5366"
	if concert.ReadMoreURL != expectedReadMoreURL {
		t.Errorf("Expected ReadMoreURL %q, got %q", expectedReadMoreURL, concert.ReadMoreURL)
	}

	expectedImageURL := "https://estonia.concert.ee/wp-content/uploads/2026/04/555x800px_Haydn_Loomine_yld-82x118.jpg"
	if concert.ImageURL != expectedImageURL {
		t.Errorf("Expected ImageURL %q, got %q", expectedImageURL, concert.ImageURL)
	}

	expectedTicketPrice := "Paid"
	if concert.TicketPrice != expectedTicketPrice {
		t.Errorf("Expected TicketPrice %q, got %q", expectedTicketPrice, concert.TicketPrice)
	}

	expectedSource := "concert"
	if concert.Source != expectedSource {
		t.Errorf("Expected Source %q, got %q", expectedSource, concert.Source)
	}
}

func TestParseDetail(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "..", "testdata", "concert_detail_fixture.html")
	file, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("Failed to open fixture file: %v", err)
	}
	defer file.Close()

	extDesc, err := ParseDetail(file)
	if err != nil {
		t.Fatalf("Failed to parse detail: %v", err)
	}

	expectedExtDesc := "Kavas:Telemann. Fantaasia nr 5 A-duur\nAugustin Hadelich on üks tänapäeva silmapaistvamaid viiuldajaid."
	if extDesc != expectedExtDesc {
		t.Errorf("Expected Extended Description %q, got %q", expectedExtDesc, extDesc)
	}
}
