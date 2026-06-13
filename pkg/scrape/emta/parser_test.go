package emta

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCalendar(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "..", "testdata", "calendar_fixture.html")
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

	expectedID := 18222
	if concert.ID != expectedID {
		t.Errorf("Expected ID %d, got %d", expectedID, concert.ID)
	}

	expectedTitle := "Jazz akadeemias. Jazzmuusika eriala avalikud kontserteksamid"
	if concert.Title != expectedTitle {
		t.Errorf("Expected Title %q, got %q", expectedTitle, concert.Title)
	}

	expectedDate := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	if !concert.Date.Equal(expectedDate) {
		t.Errorf("Expected Date %v, got %v", expectedDate, concert.Date)
	}

	expectedTime := "kell 10:00"
	if concert.RawTime != expectedTime {
		t.Errorf("Expected RawTime %q, got %q", expectedTime, concert.RawTime)
	}

	expectedVenue := "Black box"
	if concert.Venue != expectedVenue {
		t.Errorf("Expected Venue %q, got %q", expectedVenue, concert.Venue)
	}

	expectedDescription := "Magistriõppe I aasta eksamid:\nFilipp Lepalaan (löökpillid), Jaan Mesi (trompet), Vladimir Todurov (klahvpillid), Ott Ajaots (kitarr)"
	if concert.Description != expectedDescription {
		t.Errorf("Expected Description %q, got %q", expectedDescription, concert.Description)
	}

	expectedReadMoreURL := "https://emtasaalid.ee/uritused/jazz-akadeemias-jazzmuusika-eriala-avalikud-kontserteksamid-6/"
	if concert.ReadMoreURL != expectedReadMoreURL {
		t.Errorf("Expected ReadMoreURL %q, got %q", expectedReadMoreURL, concert.ReadMoreURL)
	}

	expectedImageURL := "https://emtasaalid.ee/wp-content/uploads/2026/04/Koduleht-2-2-1280x830.jpg"
	if concert.ImageURL != expectedImageURL {
		t.Errorf("Expected ImageURL %q, got %q", expectedImageURL, concert.ImageURL)
	}

	expectedTicketPrice := "Free"
	if concert.TicketPrice != expectedTicketPrice {
		t.Errorf("Expected TicketPrice %q, got %q", expectedTicketPrice, concert.TicketPrice)
	}
}

func TestParseDetail(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "..", "testdata", "detail_fixture.html")
	file, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("Failed to open fixture file: %v", err)
	}
	defer file.Close()

	extDesc, err := ParseDetail(file)
	if err != nil {
		t.Fatalf("Failed to parse detail: %v", err)
	}

	expectedExtDesc := "Ajakava:\n10.00 Filipp Lepalaan (löökpillid)\nAlates Eesti Muusika- ja Teatriakadeemia..."
	if extDesc != expectedExtDesc {
		t.Errorf("Expected Extended Description %q, got %q", expectedExtDesc, extDesc)
	}
}
