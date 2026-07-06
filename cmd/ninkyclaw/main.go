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
	"ninkyclaw/pkg/scrape/concert"
	"ninkyclaw/pkg/scrape/eccm"
	"ninkyclaw/pkg/scrape/emta"
	"ninkyclaw/pkg/scrape/filharmoonia"
	"ninkyclaw/pkg/scrape/muba"
	"ninkyclaw/pkg/scrape/phillyjoes"
)

func main() {
	now := time.Now()

	yearFlag := flag.Int("year", now.Year(), "Year to scrape calendar for")
	monthFlag := flag.Int("month", int(now.Month()), "Month to scrape calendar for (1-12)")
	rulesFlag := flag.String("rules", "", "Path to CSV file containing rating rules (keyword,rating)")
	sourceFlag := flag.String("source", "all", "Scraper source: 'all', 'emta', 'concert', 'filharmoonia', 'muba', 'eccm', or 'phillyjoes'")

	flag.Parse()

	if *monthFlag < 1 || *monthFlag > 12 {
		log.Fatalf("Invalid month: %d. Must be between 1 and 12.", *monthFlag)
	}

	var concerts []model.Concert

	log.Printf("Scraping %s calendar for %d/%02d...\n", *sourceFlag, *yearFlag, *monthFlag)

	runEMTA := *sourceFlag == "all" || *sourceFlag == "emta"
	runConcert := *sourceFlag == "all" || *sourceFlag == "concert"
	runFilharmoonia := *sourceFlag == "all" || *sourceFlag == "filharmoonia"
	runMUBA := *sourceFlag == "all" || *sourceFlag == "muba"
	runECCM := *sourceFlag == "all" || *sourceFlag == "eccm"
	runPhillyJoes := *sourceFlag == "all" || *sourceFlag == "phillyjoes"

	if *sourceFlag != "all" && *sourceFlag != "emta" && *sourceFlag != "concert" && *sourceFlag != "filharmoonia" && *sourceFlag != "muba" && *sourceFlag != "eccm" && *sourceFlag != "phillyjoes" {
		log.Fatalf("Unknown source: %s. Supported sources: all, emta, concert, filharmoonia, muba, eccm, phillyjoes.", *sourceFlag)
	}

	if runEMTA {
		log.Println("Scraping EMTA...")
		scraper := emta.NewScraper()
		emtaConcerts, err := scraper.Scrape(*yearFlag, *monthFlag)
		if err != nil {
			log.Printf("Error scraping EMTA: %v", err)
		} else {
			concerts = append(concerts, emtaConcerts...)
		}
	}

	if runConcert {
		log.Println("Scraping concert.ee...")
		scraper := concert.NewScraper()
		concertConcerts, err := scraper.Scrape(*yearFlag, *monthFlag)
		if err != nil {
			log.Printf("Error scraping concert.ee: %v", err)
		} else {
			concerts = append(concerts, concertConcerts...)
		}
	}

	if runFilharmoonia {
		log.Println("Scraping filharmoonia.ee...")
		scraper := filharmoonia.NewScraper()
		filharmooniaConcerts, err := scraper.Scrape(*yearFlag, *monthFlag)
		if err != nil {
			log.Printf("Error scraping filharmoonia.ee: %v", err)
		} else {
			concerts = append(concerts, filharmooniaConcerts...)
		}
	}

	if runMUBA {
		log.Println("Scraping muba.edu.ee...")
		scraper := muba.NewScraper()
		mubaConcerts, err := scraper.Scrape(*yearFlag, *monthFlag)
		if err != nil {
			log.Printf("Error scraping muba.edu.ee: %v", err)
		} else {
			concerts = append(concerts, mubaConcerts...)
		}
	}

	if runECCM {
		log.Println("Scraping eccm.ee...")
		scraper := eccm.NewScraper()
		eccmConcerts, err := scraper.Scrape(*yearFlag, *monthFlag)
		if err != nil {
			log.Printf("Error scraping eccm.ee: %v", err)
		} else {
			concerts = append(concerts, eccmConcerts...)
		}
	}

	if runPhillyJoes {
		log.Println("Scraping phillyjoes.com...")
		scraper := phillyjoes.NewScraper()
		pjConcerts, err := scraper.Scrape(*yearFlag, *monthFlag)
		if err != nil {
			log.Printf("Error scraping phillyjoes.com: %v", err)
		} else {
			concerts = append(concerts, pjConcerts...)
		}
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

	// Sort concerts by date ascending (chronological)
	sort.Slice(concerts, func(i, j int) bool {
		if !concerts[i].Date.Equal(concerts[j].Date) {
			return concerts[i].Date.Before(concerts[j].Date)
		}
		if concerts[i].RawTime != concerts[j].RawTime {
			return concerts[i].RawTime < concerts[j].RawTime
		}
		return rating.Priority(concerts[i].Rating) > rating.Priority(concerts[j].Rating)
	})

	outputTable(concerts)
}

func outputTable(concerts []model.Concert) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, "DATE\tTIME\tSOURCE\tRATING\tTITLE\tTICKET\tREAD MORE")
	fmt.Fprintln(w, "----\t----\t------\t------\t-----\t------\t---------")

	for _, c := range concerts {
		dateStr := c.Date.Format("2006-01-02")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			dateStr,
			c.RawTime,
			c.Source,
			c.Rating,
			c.Title,
			c.TicketPrice,
			c.ReadMoreURL,
		)
	}
	w.Flush()
}
