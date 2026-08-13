package concert

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ninkyclaw/pkg/client"
	"ninkyclaw/pkg/model"

	"github.com/PuerkitoBio/goquery"
)

const (
	ajaxURL = "https://estonia.concert.ee/wp-admin/admin-ajax.php"
)

// Scraper implements scrape.Scraper for the concert.ee calendar.
type Scraper struct {
	client *client.Client
}

// NewScraper creates a new concert.ee Scraper.
func NewScraper() *Scraper {
	return &Scraper{
		client: client.NewClient(),
	}
}

// Scrape fetches events for the given year/month via the AJAX endpoint.
func (s *Scraper) Scrape(year, month int) ([]model.Concert, error) {
	formData := ajaxFormData(year, month)
	body, err := s.client.Post(ajaxURL, formData)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events via AJAX: %w", err)
	}
	defer body.Close()

	concerts, err := s.Parse(body, year, month)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AJAX response: %w", err)
	}

	for i := range concerts {
		if concerts[i].ReadMoreURL != "" {
			detailBody, err := s.client.Fetch(concerts[i].ReadMoreURL)
			if err != nil {
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

func ajaxFormData(year, month int) string {
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)
	return fmt.Sprintf("action=ek_list_display&ek_ajax_start%%5B%%5D=%s&ek_ajax_end%%5B%%5D=%s&ek_ajax_location%%5B%%5D=Tallinn",
		firstDay.Format("2006-01-02"), lastDay.Format("2006-01-02"))
}

// Parse extracts concerts from the HTML response.
func (s *Scraper) Parse(r io.Reader, year, month int) ([]model.Concert, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	var concerts []model.Concert
	var ekIDRegex = regexp.MustCompile(`[?&]ek_id=(\d+)`)

	doc.Find(".row").Each(func(i int, sel *goquery.Selection) {
		// Find elements
		infoSel := sel.Find(".info")
		if infoSel.Length() == 0 {
			return
		}

		h2Sel := infoSel.Find("h2 a")
		title := strings.TrimSpace(h2Sel.Text())
		if title == "" {
			return
		}

		readMoreURL, _ := h2Sel.Attr("href")

		var concertID int
		if readMoreURL != "" {
			if matches := ekIDRegex.FindStringSubmatch(readMoreURL); len(matches) > 1 {
				if id, err := strconv.Atoi(matches[1]); err == nil {
					concertID = id
				}
			}
		}

		// Background image
		var imageURL string
		if imgStyle, exists := sel.Find(".image").Attr("style"); exists {
			// Extract url from style="background-image: url(...)"
			if start := strings.Index(imgStyle, "url("); start != -1 {
				end := strings.Index(imgStyle[start:], ")")
				if end != -1 {
					imgURL := imgStyle[start+4 : start+end]
					imgURL = strings.Trim(imgURL, `"'()`)
					imageURL = imgURL
				}
			}
		}

		// Date & Time
		// Format: 19.12
		dateText := strings.TrimSpace(infoSel.Find(".event-date").Text())
		// Clean up e.g., "Laupäev, 19.12"
		if commaIdx := strings.Index(dateText, ","); commaIdx != -1 {
			dateText = strings.TrimSpace(dateText[commaIdx+1:])
		}

		// Parse date (day and month)
		var parsedDate time.Time
		parts := strings.Split(dateText, ".")
		if len(parts) >= 2 {
			dayVal, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
			monthVal, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			if dayVal > 0 && monthVal > 0 {
				parsedDate = time.Date(year, time.Month(monthVal), dayVal, 0, 0, 0, 0, time.UTC)
			}
		}

		timeStr := strings.TrimSpace(infoSel.Find(".event-time").Text())
		venue := strings.TrimSpace(infoSel.Find(".event-venue").Text())

		// Ticket info
		ticketPrice := "Unknown"
		ticketURL := ""
		buyBtn := sel.Find(".col-2 a.btn.large")
		if buyBtn.Length() > 0 {
			ticketURL, _ = buyBtn.Attr("href")
			btnText := strings.ToLower(buyBtn.Text())
			if strings.Contains(btnText, "tasuta") || strings.Contains(btnText, "vaba") {
				ticketPrice = "Free"
			} else {
				ticketPrice = "Paid"
			}
		}

		concerts = append(concerts, model.Concert{
			ID:          concertID,
			Title:       title,
			Date:        parsedDate,
			RawTime:     timeStr,
			Venue:       venue,
			ReadMoreURL: readMoreURL,
			ImageURL:    imageURL,
			TicketPrice: ticketPrice,
			TicketURL:   ticketURL,
			Source:      "concert",
		})
	})

	return concerts, nil
}

// ParseDetail parses the detail page HTML.
func ParseDetail(r io.Reader) (string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return "", err
	}

	var paragraphs []string
	doc.Find(".about .text p").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	})

	return strings.Join(paragraphs, "\n"), nil
}
