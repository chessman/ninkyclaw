package filharmoonia

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"strings"
	"time"

	"ninkyclaw/pkg/client"
	"ninkyclaw/pkg/model"

	"github.com/PuerkitoBio/goquery"
)

const (
	baseURL   = "https://www.filharmoonia.ee/sundmused"
	detailURL = "https://www.filharmoonia.ee/event-details/"
)

// Scraper implements scrape.Scraper for filharmoonia.ee.
type Scraper struct {
	client *client.Client
}

// NewScraper creates a new filharmoonia Scraper.
func NewScraper() *Scraper {
	return &Scraper{
		client: client.NewClient(),
	}
}

// WixEventLocation represents location metadata.
type WixEventLocation struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// WixEventSchedulingConfig holds the start and end dates.
type WixEventSchedulingConfig struct {
	StartDate string `json:"startDate"`
}

// WixEventScheduling represents scheduling metadata.
type WixEventScheduling struct {
	Config             WixEventSchedulingConfig `json:"config"`
	StartTimeFormatted string                   `json:"startTimeFormatted"`
}

// WixEventMainImage holds the main event image URL.
type WixEventMainImage struct {
	URL string `json:"url"`
}

// WixEventTicketing represents ticketing state.
type WixEventTicketing struct {
	SoldOut bool `json:"soldOut"`
}

// WixEventRegistration holds registration data.
type WixEventRegistration struct {
	Type      int               `json:"type"`
	Ticketing WixEventTicketing `json:"ticketing"`
}

// WixEvent represents a single event in the Wix payload.
type WixEvent struct {
	ID           string               `json:"id"`
	Title        string               `json:"title"`
	Description  string               `json:"description"`
	Location     WixEventLocation     `json:"location"`
	Scheduling   WixEventScheduling   `json:"scheduling"`
	MainImage    WixEventMainImage    `json:"mainImage"`
	Slug         string               `json:"slug"`
	Registration WixEventRegistration `json:"registration"`
}

// WixWidgetComp represents the widget component structure.
type WixWidgetComp struct {
	Events struct {
		Events []WixEvent `json:"events"`
	} `json:"events"`
}

// WixAppsWarmupData maps App IDs to widget component data.
type WixAppsWarmupData map[string]map[string]json.RawMessage

// WixWarmupData represents the outer Wix warmup data JSON structure.
type WixWarmupData struct {
	AppsWarmupData WixAppsWarmupData `json:"appsWarmupData"`
}

// Scrape fetches events from the filharmoonia calendar page, parses all events,
// and filters them by the requested year and month.
func (s *Scraper) Scrape(year, month int) ([]model.Concert, error) {
	body, err := s.client.Fetch(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch filharmoonia events page: %w", err)
	}
	defer body.Close()

	allConcerts, err := s.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse filharmoonia page: %w", err)
	}

	// Filter by year and month
	var filtered []model.Concert
	for _, c := range allConcerts {
		if c.Date.Year() == year && int(c.Date.Month()) == month {
			filtered = append(filtered, c)
		}
	}

	return filtered, nil
}

// Parse extracts concerts from the filharmoonia HTML page.
func (s *Scraper) Parse(r io.Reader) ([]model.Concert, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	var scriptContent string
	doc.Find("script").Each(func(i int, sel *goquery.Selection) {
		text := sel.Text()
		if strings.Contains(text, "appsWarmupData") {
			scriptContent = text
		}
	})

	if scriptContent == "" {
		return nil, fmt.Errorf("failed to find appsWarmupData script tag")
	}

	var wixData WixWarmupData
	if err := json.Unmarshal([]byte(scriptContent), &wixData); err != nil {
		return nil, fmt.Errorf("failed to parse wix warmup data: %w", err)
	}

	var events []WixEvent
	for _, appData := range wixData.AppsWarmupData {
		for compID, rawMsg := range appData {
			if strings.HasPrefix(compID, "widgetcomp-") {
				var comp WixWidgetComp
				if err := json.Unmarshal(rawMsg, &comp); err == nil && len(comp.Events.Events) > 0 {
					events = comp.Events.Events
					break
				}
			}
		}
		if len(events) > 0 {
			break
		}
	}

	var concerts []model.Concert
	for _, event := range events {
		// Parse date
		var parsedDate time.Time
		if event.Scheduling.Config.StartDate != "" {
			parsedDate, _ = time.Parse(time.RFC3339, event.Scheduling.Config.StartDate)
			// Convert to local timezone if needed, but we keep UTC date representation
		}

		// Generate ID from event UUID
		h := fnv.New32a()
		h.Write([]byte(event.ID))
		concertID := int(h.Sum32())

		// Read more link
		readMoreURL := ""
		if event.Slug != "" {
			readMoreURL = detailURL + event.Slug
		}

		// Ticket pricing status
		ticketPrice := "Paid"
		if event.Registration.Ticketing.SoldOut {
			ticketPrice = "Sold Out"
		}

		concerts = append(concerts, model.Concert{
			ID:          concertID,
			Title:       event.Title,
			Date:        parsedDate,
			RawTime:     event.Scheduling.StartTimeFormatted,
			Venue:       event.Location.Name,
			Description: event.Description,
			ReadMoreURL: readMoreURL,
			ImageURL:    event.MainImage.URL,
			TicketPrice: ticketPrice,
			TicketURL:   readMoreURL,
			Source:      "filharmoonia",
		})
	}

	return concerts, nil
}
