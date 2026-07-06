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

const (
	ajaxURL = "https://emtasaalid.ee/wp-admin/admin-ajax.php"
)

// ajaxFormData builds the POST body for the EMTA AJAX calendar endpoint.
func ajaxFormData(year, month int) string {
	return fmt.Sprintf("action=myplugin_ajax_load&month=%02d&year=%d", month, year)
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

// Scrape fetches events for the given year/month via the AJAX endpoint that the
// site's JavaScript uses, parses the HTML fragment, then enriches each concert
// with its extended description from the detail page.
func (s *Scraper) Scrape(year, month int) ([]model.Concert, error) {
	// The static calendar page always returns the current month regardless of URL
	// params — those are only read client-side by JS. We POST directly to the same
	// AJAX endpoint the JS calls to get the correct month's event list.
	body, err := s.client.Post(ajaxURL, ajaxFormData(year, month))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events via AJAX: %w", err)
	}
	defer body.Close()

	concerts, err := s.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AJAX response: %w", err)
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

	// The AJAX endpoint returns bare .event fragments (no .event-list wrapper),
	// so we match .event directly which also works for the full calendar page.
	doc.Find(".event").Each(func(i int, s *goquery.Selection) {
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
			Source:      "emta",
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
