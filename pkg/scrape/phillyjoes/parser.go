package phillyjoes

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
	collectionID = "5b56dc02f950b7329f323f1a"
	apiURL       = "https://www.phillyjoes.com/api/open/GetItemsByMonth"
	domain       = "https://www.phillyjoes.com"
)

var monthNames = map[int]string{
	1:  "January",
	2:  "February",
	3:  "March",
	4:  "April",
	5:  "May",
	6:  "June",
	7:  "July",
	8:  "August",
	9:  "September",
	10: "October",
	11: "November",
	12: "December",
}

// Scraper implements scrape.Scraper for phillyjoes.com.
type Scraper struct {
	client *client.Client
}

// NewScraper creates a new phillyjoes Scraper.
func NewScraper() *Scraper {
	return &Scraper{
		client: client.NewClient(),
	}
}

type PhillyJoesLocation struct {
	AddressTitle string `json:"addressTitle"`
}

// PhillyJoesStructuredContent holds the dates. Squarespace used to put these at
// the top level of the item; they live in structuredContent now.
type PhillyJoesStructuredContent struct {
	StartDate int64 `json:"startDate"` // Epoch milliseconds
	EndDate   int64 `json:"endDate"`   // Epoch milliseconds
}

type PhillyJoesEvent struct {
	Title             string                      `json:"title"`
	FullURL           string                      `json:"fullUrl"`
	AssetURL          string                      `json:"assetUrl"`
	StructuredContent PhillyJoesStructuredContent `json:"structuredContent"`
	Location          PhillyJoesLocation          `json:"location"`
}

// Scrape fetches events for the given year/month.
func (s *Scraper) Scrape(year, month int) ([]model.Concert, error) {
	monthName, ok := monthNames[month]
	if !ok {
		return nil, fmt.Errorf("invalid month: %d", month)
	}

	monthParam := fmt.Sprintf("%s-%d", monthName, year)
	url := fmt.Sprintf("%s?month=%s&collectionId=%s", apiURL, monthParam, collectionID)

	body, err := s.client.Fetch(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch phillyjoes calendar: %w", err)
	}
	defer body.Close()

	allEvents, err := s.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse phillyjoes calendar: %w", err)
	}

	// Filter by exact month/year (the API sometimes returns boundary events from adjacent months)
	var concerts []model.Concert
	for _, event := range allEvents {
		if event.Date.Year() == year && int(event.Date.Month()) == month {
			concerts = append(concerts, event)
		}
	}

	// Enrich details
	for i := range concerts {
		if concerts[i].ReadMoreURL != "" {
			detailBody, err := s.client.Fetch(concerts[i].ReadMoreURL)
			if err != nil {
				continue
			}
			extDesc, ticketPrice, ticketURL, err := ParseDetail(detailBody)
			detailBody.Close()
			if err == nil {
				if extDesc != "" {
					concerts[i].Description = extDesc
					concerts[i].ExtendedDescription = extDesc
				}
				if ticketPrice != "" {
					concerts[i].TicketPrice = ticketPrice
				}
				if ticketURL != "" {
					concerts[i].TicketURL = ticketURL
				}
			}
			time.Sleep(200 * time.Millisecond) // Polite sleep
		}
	}

	return concerts, nil
}

// Parse parses the Squarespace JSON event list.
func (s *Scraper) Parse(r io.Reader) ([]model.Concert, error) {
	var events []PhillyJoesEvent
	if err := json.NewDecoder(r).Decode(&events); err != nil {
		return nil, err
	}

	// Philly Joe's is in Tallinn – convert UTC epoch to local time to get the
	// correct date and clock time for display.
	loc, err := time.LoadLocation("Europe/Tallinn")
	if err != nil {
		// Fallback to UTC+3 fixed zone when tzdata is not available
		loc = time.FixedZone("EET", 3*60*60)
	}

	var concerts []model.Concert
	for _, event := range events {
		date := time.UnixMilli(event.StructuredContent.StartDate).In(loc)

		// Read More URL
		readMoreURL := ""
		if event.FullURL != "" {
			if strings.HasPrefix(event.FullURL, "/") {
				readMoreURL = domain + event.FullURL
			} else {
				readMoreURL = event.FullURL
			}
		}

		// Create deterministic ID. The item carries no event id any more, and its
		// systemDataId is the image's, shared by every event that reuses a poster,
		// so the event page URL is the one unique stable string left.
		h := fnv.New32a()
		h.Write([]byte(readMoreURL))
		concertID := int(h.Sum32())

		// Venue
		venue := event.Location.AddressTitle
		if venue == "" {
			venue = "Philly Joe's"
		}

		// Show local clock time, not UTC
		rawTime := date.Format("15:04")

		concerts = append(concerts, model.Concert{
			ID:          concertID,
			Title:       event.Title,
			Date:        time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC),
			RawTime:     rawTime,
			Venue:       venue,
			ReadMoreURL: readMoreURL,
			ImageURL:    event.AssetURL,
			TicketPrice: "Paid", // default
			Source:      "phillyjoes",
		})
	}

	return concerts, nil
}

// ParseDetail parses the HTML detail page of a Philly Joe's event.
func ParseDetail(r io.Reader) (string, string, string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return "", "", "", err
	}

	var paragraphs []string
	doc.Find("main#page .sqs-html-content p").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	})
	description := strings.Join(paragraphs, "\n")

	ticketPrice := "Paid" // Default is paid
	ticketURL := ""

	// Look for button block links (e.g. TICKETS or PILETID)
	doc.Find(".sqs-block-button-container a").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		btnText := strings.ToLower(strings.TrimSpace(s.Text()))
		if href != "" && (strings.Contains(btnText, "ticket") || strings.Contains(btnText, "pilet") || strings.Contains(btnText, "fienta") || strings.Contains(btnText, "keskus")) {
			ticketURL = href
		}
	})

	// Fallback to description scanning for ticket URLs if none found in button blocks
	if ticketURL == "" {
		words := strings.Fields(description)
		for _, w := range words {
			if strings.Contains(w, "piletikeskus.ee") || strings.Contains(w, "fienta.com") {
				ticketURL = strings.Trim(w, `()[]{}.,"';:`)
				break
			}
		}
	}

	if ticketURL == "" {
		// Only check if it's explicitly free when no ticket URL exists
		descLower := strings.ToLower(description)
		if strings.Contains(descLower, "tasuta") || strings.Contains(descLower, "vaba sissepääs") || strings.Contains(descLower, "free entrance") {
			ticketPrice = "Free"
		}
	}

	return description, ticketPrice, ticketURL, nil
}
