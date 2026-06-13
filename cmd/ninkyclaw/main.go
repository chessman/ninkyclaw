package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"ninkyclaw/pkg/model"
	"ninkyclaw/pkg/rating"
	"ninkyclaw/pkg/scrape/emta"
)

func main() {
	now := time.Now()

	yearFlag := flag.Int("year", now.Year(), "Year to scrape calendar for")
	monthFlag := flag.Int("month", int(now.Month()), "Month to scrape calendar for (1-12)")
	rulesFlag := flag.String("rules", "", "Path to CSV file containing rating rules (keyword,rating)")

	flag.Parse()

	if *monthFlag < 1 || *monthFlag > 12 {
		log.Fatalf("Invalid month: %d. Must be between 1 and 12.", *monthFlag)
	}

	log.Printf("Scraping EMTA calendar for %d/%02d...\n", *yearFlag, *monthFlag)
	scraper := emta.NewScraper()
	concerts, err := scraper.Scrape(*yearFlag, *monthFlag)
	if err != nil {
		log.Fatalf("Error scraping: %v", err)
	}
	log.Printf("Successfully scraped %d concerts.\n", len(concerts))

	if *rulesFlag != "" {
		log.Printf("Loading rating rules from %s...\n", *rulesFlag)
		rater, err := rating.NewKeywordRaterFromCSVFile(*rulesFlag)
		if err != nil {
			log.Fatalf("Error loading rules file: %v", err)
		}

		for i := range concerts {
			r, err := rater.Rate(concerts[i])
			if err != nil {
				log.Printf("Warning: failed to rate concert %q: %v", concerts[i].Title, err)
				continue
			}
			concerts[i].Rating = r.String()
		}
	} else {
		// If no rules CSV is loaded, set rating to Very Low so they sort consistently
		for i := range concerts {
			concerts[i].Rating = rating.VeryLow.String()
		}
	}

	// Sort concerts by rating highest to lowest (descending priority)
	sort.Slice(concerts, func(i, j int) bool {
		return rating.Priority(concerts[i].Rating) > rating.Priority(concerts[j].Rating)
	})

	outputTable(concerts)
}

func outputTable(concerts []model.Concert) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, "DATE\tTIME\tVENUE\tTITLE\tTICKET\tRATING\tREAD MORE")
	fmt.Fprintln(w, "----\t----\t-----\t-----\t------\t------\t---------")

	for _, c := range concerts {
		dateStr := c.Date.Format("2006-01-02")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			dateStr,
			c.RawTime,
			c.Venue,
			c.Title,
			c.TicketPrice,
			c.Rating,
			c.ReadMoreURL,
		)
	}
	w.Flush()
}
