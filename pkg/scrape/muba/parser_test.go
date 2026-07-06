package muba

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	htmlData := `
<div class="event-card position-relative"> 
    <div class="event-card-wrapper d-flex flex-column flex-lg-row">      
        <div class="event-date d-flex align-items-start">
            <div class="bg-black d-flex py-2 px-4">
                                    <p class="date m-0">
                        <span class="weekday">P,</span>
                        <span class="day h2 d-block">10.</span>
                        <span class="month">mai</span>
                    </p>
                            </div>
        </div>        
        <div class="event-content pt-0 pt-lg-4 with-image pe-lg-10 pb-6 ps-lg-6">
            <a href="https://muba.edu.ee/sundmused/iii-muba-saxfest/" class="item-card-link text-decoration-none d-block"></a>
            <h3 class="event-card-title m-0">III MUBA SaxFest</h3>
            <div class="event-details d-lg-flex flex-wrap mt-2">
                <p class="event-location">MUBA Suur saal</p>
                <p class="event-time">17:00 – 18:00</p>
            </div>
        </div> 
        <div class="event-img">
            <img src="https://muba.edu.ee/image.jpg" />
        </div>
    </div>
</div>
<nav class="navigation pagination">
    <div class="nav-links">
        <a class="next page-numbers" href="https://muba.edu.ee/sundmused/page/2/">Next</a>
    </div>
</nav>
`

	scraper := NewScraper()
	concerts, nextURL, err := scraper.Parse(strings.NewReader(htmlData), 2026, 5)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(concerts) != 1 {
		t.Fatalf("Expected 1 concert, got %d", len(concerts))
	}

	c := concerts[0]
	if c.Title != "III MUBA SaxFest" {
		t.Errorf("Expected Title 'III MUBA SaxFest', got %q", c.Title)
	}

	if c.Venue != "MUBA Suur saal" {
		t.Errorf("Expected Venue 'MUBA Suur saal', got %q", c.Venue)
	}

	if c.RawTime != "17:00 – 18:00" {
		t.Errorf("Expected RawTime '17:00 – 18:00', got %q", c.RawTime)
	}

	if c.ReadMoreURL != "https://muba.edu.ee/sundmused/iii-muba-saxfest/" {
		t.Errorf("Expected ReadMoreURL 'https://muba.edu.ee/sundmused/iii-muba-saxfest/', got %q", c.ReadMoreURL)
	}

	if c.ImageURL != "https://muba.edu.ee/image.jpg" {
		t.Errorf("Expected ImageURL 'https://muba.edu.ee/image.jpg', got %q", c.ImageURL)
	}

	expectedDate := "2026-05-10"
	if c.Date.Format("2006-01-02") != expectedDate {
		t.Errorf("Expected Date %s, got %s", expectedDate, c.Date.Format("2006-01-02"))
	}

	if nextURL != "https://muba.edu.ee/sundmused/page/2/" {
		t.Errorf("Expected nextURL 'https://muba.edu.ee/sundmused/page/2/', got %q", nextURL)
	}
}

func TestParseDetail(t *testing.T) {
	detailHTML := `
<main class="single-event">
    <div class="regular-content">
        <div class="content-column-classes">
            <p>Esinevad solistid, ansamblid ja saksofoniorkester</p>
            <div class="wp-block-button">
                <a class="wp-block-button__link wp-element-button" href="https://fienta.com/et/iii-muba-saxfest">TASUTA PILET FIENTAST</a>
            </div>
        </div>
    </div>
</main>
`

	desc, ticketPrice, ticketURL, err := ParseDetail(strings.NewReader(detailHTML))
	if err != nil {
		t.Fatalf("ParseDetail error: %v", err)
	}

	if desc != "Esinevad solistid, ansamblid ja saksofoniorkester" {
		t.Errorf("Expected description 'Esinevad solistid, ansamblid ja saksofoniorkester', got %q", desc)
	}

	if ticketPrice != "Free" {
		t.Errorf("Expected ticketPrice 'Free', got %q", ticketPrice)
	}

	if ticketURL != "https://fienta.com/et/iii-muba-saxfest" {
		t.Errorf("Expected ticketURL 'https://fienta.com/et/iii-muba-saxfest', got %q", ticketURL)
	}
}
