package emta

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"ninkyclaw/pkg/client"
	"ninkyclaw/pkg/model"

	"github.com/PuerkitoBio/goquery"
)

const baseURL = "https://emtasaalid.ee/kalender/"

// CalendarURL builds the EMTA calendar URL for the given year and month.
func CalendarURL(year, month int) string {
	return fmt.Sprintf("%s?aasta=%d&kuu=%d", baseURL, year, month)
}

// Scraper implements scrape.Scraper for the EMTA calendar.
type Scraper struct {
	client *client.Client
}

// NewScraper creates a new EMTA Scraper.
func NewScraper() *Scraper {
	return &Scraper{
		client: client.NewClient(),
	}
}

// Scrape fetches the calendar page, parses it, fetches all detail pages, and returns the concerts.
func (s *Scraper) Scrape(year, month int) ([]model.Concert, error) {
	url := CalendarURL(year, month)
	body, err := s.client.Fetch(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch calendar page: %w", err)
	}
	defer body.Close()

	concerts, err := s.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse calendar page: %w", err)
	}

	for i := range concerts {
		if concerts[i].ReadMoreURL != "" {
			detailBody, err := s.client.Fetch(concerts[i].ReadMoreURL)
			if err != nil {
				// Log or skip, but continue scraping others
				continue
			}
			extDesc, err := ParseDetail(detailBody)
			detailBody.Close()
			if err == nil {
				concerts[i].ExtendedDescription = extDesc
			}
			time.Sleep(300 * time.Millisecond) // Polite sleep to prevent spamming
		}
	}

	return concerts, nil
}

// Parse reads HTML content from the EMTA calendar page and extracts concerts.
func (s *Scraper) Parse(r io.Reader) ([]model.Concert, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	var concerts []model.Concert

	doc.Find(".event-list .event").Each(func(i int, s *goquery.Selection) {
		// Extract persistent ID from class attribute (e.g., "post-18222")
		var concertID int
		if classVal, exists := s.Attr("class"); exists {
			for _, class := range strings.Fields(classVal) {
				if strings.HasPrefix(class, "post-") {
					idStr := strings.TrimPrefix(class, "post-")
					if val, err := strconv.Atoi(idStr); err == nil {
						concertID = val
					}
					break
				}
			}
		}

		// Extract raw date string (e.g. 2026/6/9)
		rawDate, _ := s.Attr("data-date")
		var parsedDate time.Time
		if rawDate != "" {
			parsedDate, _ = time.Parse("2006/1/2", rawDate)
		}

		details := s.Find(".details")
		title := strings.TrimSpace(details.Find("h2").First().Text())

		timeStr := strings.TrimSpace(details.Find("span.time").Text())
		timeStr = strings.TrimPrefix(timeStr, "kell ")
		venue := strings.TrimSpace(details.Find("span.cats i").Text())

		// Gather description paragraphs from the desktop '.info' container
		var descParas []string
		s.Find(".info a p").Each(func(_ int, pSel *goquery.Selection) {
			descParas = append(descParas, strings.TrimSpace(pSel.Text()))
		})
		description := strings.Join(descParas, "\n")

		// Extract "Read More" URL
		var readMoreURL string
		readMoreSel := s.Find(".show-for-large .buttons-row a.read-more")
		if readMoreSel.Length() > 0 {
			readMoreURL, _ = readMoreSel.Attr("href")
		}

		// Extract image/poster URL from data-bgset
		var imageURL string
		bgset, exists := s.Find("a.image-container .image").Attr("data-bgset")
		if exists && bgset != "" {
			parts := strings.Split(bgset, ",")
			if len(parts) > 0 {
				firstPart := strings.TrimSpace(parts[0])
				urlParts := strings.Fields(firstPart)
				if len(urlParts) > 0 {
					imageURL = urlParts[0]
				}
			}
		}

		// Retrieve ticket details
		ticketPrice := "Unknown"
		ticketURL := ""

		// Find button inside buttons-row that is NOT the read-more button
		ticketBtn := s.Find(".show-for-large .buttons-row a.button").Not(".read-more")
		if ticketBtn.Length() > 0 {
			ticketURL, _ = ticketBtn.Attr("href")
			btnText := strings.ToLower(ticketBtn.Text())
			if strings.Contains(btnText, "tasuta") || strings.Contains(btnText, "vaba sissepääs") {
				ticketPrice = "Free"
			} else {
				ticketPrice = "Paid"
			}
		} else {
			ticketSpan := s.Find(".show-for-large .buttons-row .ticket")
			if ticketSpan.Length() > 0 {
				spanText := strings.ToLower(ticketSpan.Text())
				if strings.Contains(spanText, "tasuta") {
					ticketPrice = "Free"
				} else {
					ticketPrice = strings.TrimSpace(ticketSpan.First().Text())
				}
			}
		}

		concerts = append(concerts, model.Concert{
			ID:          concertID,
			Title:       title,
			Date:        parsedDate,
			RawTime:     timeStr,
			Venue:       venue,
			Description: description,
			ReadMoreURL: readMoreURL,
			ImageURL:    imageURL,
			TicketPrice: ticketPrice,
			TicketURL:   ticketURL,
		})
	})

	return concerts, nil
}

// ParseDetail reads HTML content from a concert detail page and extracts the extended description.
func ParseDetail(r io.Reader) (string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return "", err
	}

	var paragraphs []string
	doc.Find(".cell.text p").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	})

	return strings.Join(paragraphs, "\n"), nil
}
