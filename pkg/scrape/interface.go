package scrape

import (
	"ninkyclaw/pkg/model"
)

// Scraper defines a common interface for scraping events for a specific year and month.
type Scraper interface {
	Scrape(year, month int) ([]model.Concert, error)
}
