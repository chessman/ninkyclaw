package eccm

import (
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
	baseURL = "https://www.eccm.ee/index.php/et/syndmusest"
	domain  = "https://www.eccm.ee"
)

// Scraper implements scrape.Scraper for eccm.ee.
type Scraper struct {
	client *client.Client
}

// NewScraper creates a new eccm.ee Scraper.
func NewScraper() *Scraper {
	return &Scraper{
		client: client.NewClient(),
	}
}

// Scrape fetches events for the given year/month.
func (s *Scraper) Scrape(year, month int) ([]model.Concert, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)

	dateFromStr := startDate.Format("2006-01-02")
	dateToStr := endDate.Format("2006-01-02")

	nextURL := fmt.Sprintf("%s?filter_from=%s&filter_to=%s", baseURL, dateFromStr, dateToStr)
	var concerts []model.Concert

	for nextURL != "" {
		body, err := s.client.Fetch(nextURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %s: %w", nextURL, err)
		}

		pageConcerts, nextPage, err := s.Parse(body, year, month)
		body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to parse page: %w", err)
		}

		concerts = append(concerts, pageConcerts...)
		nextURL = nextPage
	}

	// Enrich with details
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

// Parse extracts list of concerts and the next page URL.
func (s *Scraper) Parse(r io.Reader, year, month int) ([]model.Concert, string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, "", err
	}

	var concerts []model.Concert

	doc.Find(".ic-list-event").Each(func(i int, sel *goquery.Selection) {
		titleSel := sel.Find(".ic-event-title h2 a").First()
		if titleSel.Length() == 0 {
			titleSel = sel.Find(".ic-event-title a").First()
		}
		title := strings.TrimSpace(titleSel.Text())
		if title == "" {
			return
		}

		readMorePath, _ := titleSel.Attr("href")
		readMoreURL := ""
		if readMorePath != "" {
			if strings.HasPrefix(readMorePath, "/") {
				readMoreURL = domain + readMorePath
			} else {
				readMoreURL = readMorePath
			}
		}

		// Parse date
		var parsedDate time.Time
		dateStr := strings.TrimSpace(sel.Find(".ic-single-startdate, .ic-single-next, .ic-period-startdate").First().Text())
		if dateStr != "" {
			parsedDate, _ = time.Parse("02.01.2006", dateStr)
		}

		// RawTime
		startTime := strings.TrimSpace(sel.Find(".ic-single-starttime, .ic-period-starttime").First().Text())
		endTime := strings.TrimSpace(sel.Find(".ic-single-endtime, .ic-period-endtime").First().Text())
		endDate := strings.TrimSpace(sel.Find(".ic-period-enddate").First().Text())

		rawTime := startTime
		if endDate != "" && endDate != dateStr {
			if endTime != "" {
				rawTime = fmt.Sprintf("%s - %s %s", startTime, endDate, endTime)
			} else {
				rawTime = fmt.Sprintf("%s - %s", startTime, endDate)
			}
		} else if startTime != "" && endTime != "" {
			rawTime = startTime + " - " + endTime
		}

		// Venue
		venue := strings.TrimSpace(sel.Find(".place.ic-place").Text())

		// Short Description
		description := strings.TrimSpace(sel.Find(".descshort.ic-descshort").Text())

		// Image
		var imageURL string
		if style, exists := sel.Find(".ic-box-date").Attr("style"); exists {
			idx := strings.Index(style, "url(")
			if idx != -1 {
				sub := style[idx+4:]
				end := strings.Index(sub, ")")
				if end != -1 {
					imgURL := strings.Trim(sub[:end], `'" `)
					if imgURL != "" {
						if strings.HasPrefix(imgURL, "/") {
							imageURL = domain + imgURL
						} else {
							imageURL = imgURL
						}
					}
				}
			}
		}

		// Deterministic ID
		h := fnv.New32a()
		h.Write([]byte(readMoreURL))
		concertID := int(h.Sum32())

		concerts = append(concerts, model.Concert{
			ID:          concertID,
			Title:       title,
			Date:        parsedDate,
			RawTime:     rawTime,
			Venue:       venue,
			Description: description,
			ReadMoreURL: readMoreURL,
			ImageURL:    imageURL,
			TicketPrice: "Free", // default, detail page will enrich
			Source:      "eccm",
		})
	})

	// Check if there is next page
	var nextURL string
	nextSel := doc.Find("div.ic-pagination div.ic-next a")
	if nextSel.Length() > 0 {
		href, _ := nextSel.Attr("href")
		if href != "" {
			if strings.HasPrefix(href, "/") {
				nextURL = domain + href
			} else {
				nextURL = href
			}
		}
	}

	return concerts, nextURL, nil
}

// ParseDetail reads HTML content from an event detail page.
func ParseDetail(r io.Reader) (string, string, string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return "", "", "", err
	}

	// Description
	var paragraphs []string
	doc.Find(".ic-full-description p").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	})
	description := strings.Join(paragraphs, "\n")

	if description == "" {
		description = strings.TrimSpace(doc.Find(".ic-full-description").Text())
	}
	if description == "" {
		description = strings.TrimSpace(doc.Find("#ic-detail-desc").Text())
	}

	// Ticket Price and URL
	ticketPrice := "Free"
	ticketURL := ""

	ticketSel := doc.Find(".ic-info-tickets a")
	if ticketSel.Length() == 0 {
		ticketSel = doc.Find(".ic-info-tickets_eng a")
	}

	if ticketSel.Length() > 0 {
		href, _ := ticketSel.Attr("href")
		if href != "" {
			ticketURL = href
			ticketPrice = "Paid"
		}
	} else {
		// Fallback: check if description mentions Fienta or other links
		descLower := strings.ToLower(description)
		if strings.Contains(descLower, "fienta.com") {
			ticketPrice = "Paid"
			words := strings.Fields(description)
			for _, w := range words {
				if strings.Contains(w, "fienta.com") {
					ticketURL = strings.Trim(w, `()[]{}.,"';:`)
					break
				}
			}
		} else if strings.Contains(descLower, "piletid") || strings.Contains(descLower, "pilet:") {
			ticketPrice = "Paid"
		}
	}

	return description, ticketPrice, ticketURL, nil
}
