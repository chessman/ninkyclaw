package muba

import (
	"fmt"
	"hash/fnv"
	"io"
	"strconv"
	"strings"
	"time"

	"ninkyclaw/pkg/client"
	"ninkyclaw/pkg/model"

	"github.com/PuerkitoBio/goquery"
)

// Scraper implements scrape.Scraper for MUBA ( Tallinna Muusika- ja Balletikool ).
type Scraper struct {
	client *client.Client
}

// NewScraper creates a new MUBA Scraper.
func NewScraper() *Scraper {
	return &Scraper{
		client: client.NewClient(),
	}
}

// parseEstonianMonth returns the month number (1-12) for Estonian month name.
func parseEstonianMonth(mStr string) int {
	mStr = strings.ToLower(strings.TrimSpace(mStr))
	mStr = strings.TrimSuffix(mStr, ".")
	switch {
	case strings.HasPrefix(mStr, "jaan"):
		return 1
	case strings.HasPrefix(mStr, "veeb"):
		return 2
	case strings.HasPrefix(mStr, "märts") || strings.HasPrefix(mStr, "marts"):
		return 3
	case strings.HasPrefix(mStr, "apr"):
		return 4
	case strings.HasPrefix(mStr, "mai"):
		return 5
	case strings.HasPrefix(mStr, "juun"):
		return 6
	case strings.HasPrefix(mStr, "juul"):
		return 7
	case strings.HasPrefix(mStr, "aug"):
		return 8
	case strings.HasPrefix(mStr, "sept"):
		return 9
	case strings.HasPrefix(mStr, "okt"):
		return 10
	case strings.HasPrefix(mStr, "nov"):
		return 11
	case strings.HasPrefix(mStr, "dets"):
		return 12
	default:
		return 0
	}
}

// Scrape fetches events for the given year/month.
func (s *Scraper) Scrape(year, month int) ([]model.Concert, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)

	dateFromStr := startDate.Format("2006-01-02")
	dateToStr := endDate.Format("2006-01-02")

	url := fmt.Sprintf("https://muba.edu.ee/sundmused/?date_from=%s&date_to=%s", dateFromStr, dateToStr)
	var concerts []model.Concert

	for url != "" {
		body, err := s.client.Fetch(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %s: %w", url, err)
		}

		pageConcerts, nextURL, err := s.Parse(body, year, month)
		body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to parse page: %w", err)
		}

		concerts = append(concerts, pageConcerts...)
		url = nextURL
	}

	// Enrich concerts with details from their single pages
	for i := range concerts {
		if concerts[i].ReadMoreURL != "" {
			detailBody, err := s.client.Fetch(concerts[i].ReadMoreURL)
			if err != nil {
				continue
			}
			extDesc, ticketPrice, ticketURL, err := ParseDetail(detailBody)
			detailBody.Close()
			if err == nil {
				concerts[i].Description = extDesc
				concerts[i].ExtendedDescription = extDesc
				if ticketPrice != "" {
					concerts[i].TicketPrice = ticketPrice
				}
				if ticketURL != "" {
					concerts[i].TicketURL = ticketURL
				}
			}
		}
	}

	return concerts, nil
}

// Parse parses the list of events from the response body.
func (s *Scraper) Parse(r io.Reader, year, month int) ([]model.Concert, string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, "", err
	}

	var concerts []model.Concert

	doc.Find(".event-card").Each(func(i int, sel *goquery.Selection) {
		title := strings.TrimSpace(sel.Find(".event-card-title").First().Text())
		if title == "" {
			return
		}

		readMoreURL, _ := sel.Find("a.item-card-link").First().Attr("href")
		imageURL, _ := sel.Find(".event-img img").First().Attr("src")
		venue := strings.TrimSpace(sel.Find(".event-location").First().Text())
		rawTime := strings.TrimSpace(sel.Find(".event-time").First().Text())

		dayStr := strings.TrimSpace(sel.Find(".event-date .day").First().Text())
		dayStr = strings.TrimSuffix(dayStr, ".")
		day, _ := strconv.Atoi(dayStr)

		monthStr := strings.TrimSpace(sel.Find(".event-date .month").First().Text())
		monthNum := parseEstonianMonth(monthStr)
		if monthNum == 0 {
			monthNum = month
		}

		var parsedDate time.Time
		if day > 0 {
			parsedDate = time.Date(year, time.Month(monthNum), day, 0, 0, 0, 0, time.UTC)
		} else {
			parsedDate = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		}

		// Generate a deterministic integer ID using FNV-32a on the slug or URL
		h := fnv.New32a()
		h.Write([]byte(readMoreURL))
		concertID := int(h.Sum32())

		concerts = append(concerts, model.Concert{
			ID:          concertID,
			Title:       title,
			Date:        parsedDate,
			RawTime:     rawTime,
			Venue:       venue,
			ReadMoreURL: readMoreURL,
			ImageURL:    imageURL,
			TicketPrice: "Free", // default
			Source:      "muba",
		})
	})

	// Get next page URL if it exists
	var nextURL string
	if nextLink := doc.Find(".pagination a.next"); nextLink.Length() > 0 {
		nextURL, _ = nextLink.Attr("href")
	}

	return concerts, nextURL, nil
}

// ParseDetail reads HTML content from an event detail page and extracts description and tickets.
func ParseDetail(r io.Reader) (string, string, string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return "", "", "", err
	}

	var paragraphs []string
	doc.Find("main.single-event .content-column-classes p").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	})
	description := strings.Join(paragraphs, "\n")

	// Find ticketing buttons
	ticketPrice := "Free"
	ticketURL := ""
	doc.Find("main.single-event .content-column-classes a.wp-element-button").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && href != "" {
			btnText := strings.ToLower(s.Text())
			if strings.Contains(btnText, "pilet") || strings.Contains(btnText, "fienta") || strings.Contains(btnText, "tasuta") {
				ticketURL = href
				if strings.Contains(btnText, "tasuta") {
					ticketPrice = "Free"
				} else {
					ticketPrice = "Paid"
				}
			}
		}
	})

	return description, ticketPrice, ticketURL, nil
}
