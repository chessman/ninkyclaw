package eccm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseCalendar(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "..", "scratch_eccm.html")
	file, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("Failed to open calendar fixture: %v", err)
	}
	defer file.Close()

	scraper := NewScraper()
	concerts, nextURL, err := scraper.Parse(file, 2026, 9)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(concerts) == 0 {
		t.Fatalf("Expected concerts, got 0")
	}

	// Verify next URL
	expectedNextURL := "https://www.eccm.ee/index.php/et/syndmusest?page=2"
	if nextURL != expectedNextURL {
		t.Errorf("Expected nextURL %q, got %q", expectedNextURL, nextURL)
	}

	// Let's test the first parsed concert
	first := concerts[0]
	if first.Title != "Schönbergi sari" {
		t.Errorf("Expected Title 'Schönbergi sari', got %q", first.Title)
	}

	expectedDate := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	if !first.Date.Equal(expectedDate) {
		t.Errorf("Expected Date %v, got %v", expectedDate, first.Date)
	}

	if first.RawTime != "19:00 - 22:00" {
		t.Errorf("Expected RawTime '19:00 - 22:00', got %q", first.RawTime)
	}

	if first.Venue != "Eesti Nüüdismuusika Keskus" {
		t.Errorf("Expected Venue 'Eesti Nüüdismuusika Keskus', got %q", first.Venue)
	}

	if first.ReadMoreURL != "https://www.eccm.ee/index.php/et/syndmusest/71-schoenbergi-sari-3/2026-09-10-19-00" {
		t.Errorf("Unexpected ReadMoreURL: %q", first.ReadMoreURL)
	}

	expectedImg := "https://www.eccm.ee/images/icagenda/thumbs/themes/ic_medium_w300h300q100_kurtag-100.jpg"
	if first.ImageURL != expectedImg {
		t.Errorf("Expected ImageURL %q, got %q", expectedImg, first.ImageURL)
	}

	if first.Source != "eccm" {
		t.Errorf("Expected Source 'eccm', got %q", first.Source)
	}
}

func TestParseDetail(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "..", "scratch_eccm_detail.html")
	file, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("Failed to open detail fixture: %v", err)
	}
	defer file.Close()

	desc, ticketPrice, ticketURL, err := ParseDetail(file)
	if err != nil {
		t.Fatalf("ParseDetail error: %v", err)
	}

	if !strings.Contains(desc, "Schönbergi sari") {
		t.Errorf("Expected description to contain 'Schönbergi sari'")
	}

	if ticketPrice != "Paid" {
		t.Errorf("Expected ticketPrice 'Paid', got %q", ticketPrice)
	}

	expectedTicketURL := "https://fienta.com/et/schoenbergi-sari-kurtag-100"
	if ticketURL != expectedTicketURL {
		t.Errorf("Expected ticketURL %q, got %q", expectedTicketURL, ticketURL)
	}
}
