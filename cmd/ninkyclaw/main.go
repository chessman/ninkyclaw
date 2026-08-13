package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
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
			r, matched, err := rater.Rate(concerts[i])
			if err != nil {
				log.Printf("Warning: failed to rate concert %q: %v", concerts[i].Title, err)
				continue
			}
			concerts[i].Rating = r.String()
			concerts[i].MatchedKeywords = matched
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

// headers doubles as the column count: every emitted line has len(headers) cells.
var headers = []string{"DATE", "TIME", "SOURCE", "RATING", "KEYWORDS", "TITLE", "TICKET", "READ MORE"}

// Column widths for the free-text columns. Everything else is short and fixed.
const (
	titleWidth   = 80
	keywordWidth = 40
)

// wrap word-wraps s into lines of at most width runes. Words longer than width
// are hard-split. Returns nil for empty input.
func wrap(s string, width int) []string {
	if width < 1 {
		width = 1
	}

	var lines []string
	var line []rune

	for _, word := range strings.Fields(s) {
		w := []rune(word)

		for len(w) > width {
			if len(line) > 0 {
				lines = append(lines, string(line))
				line = nil
			}
			lines = append(lines, string(w[:width]))
			w = w[width:]
		}

		switch {
		case len(line) == 0:
			line = w
		case len(line)+1+len(w) > width:
			lines = append(lines, string(line))
			line = w
		default:
			line = append(append(line, ' '), w...)
		}
	}

	if len(line) > 0 {
		lines = append(lines, string(line))
	}

	return lines
}

// at returns lines[i], or "" past the end, so columns of differing heights can
// be zipped into rows.
func at(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return ""
}

func outputTable(concerts []model.Concert) {
	// No TabIndent: it collapses the leading empty cells of continuation lines
	// into an indent instead of padding them to the column width.
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	rules := make([]string, len(headers))
	for i, h := range headers {
		rules[i] = strings.Repeat("-", len(h))
	}
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	fmt.Fprintln(w, strings.Join(rules, "\t"))

	for _, c := range concerts {
		for _, row := range concertRows(c) {
			fmt.Fprintln(w, strings.Join(row, "\t"))
		}
	}
	w.Flush()
}

// concertRows renders one concert as the physical lines it occupies: the wrapped
// title and keywords spill onto continuation lines, the single-line fields stay
// on the first. Every row has len(headers) cells — a short row would end
// tabwriter's column block and the columns after it would stop lining up.
func concertRows(c model.Concert) [][]string {
	title := wrap(c.Title, titleWidth)
	keywords := wrap(strings.Join(c.MatchedKeywords, ", "), keywordWidth)

	rows := make([][]string, max(len(title), len(keywords), 1))
	for i := range rows {
		rows[i] = make([]string, len(headers))
		rows[i][4], rows[i][5] = at(keywords, i), at(title, i)
	}

	rows[0][0] = c.Date.Format("2006-01-02")
	rows[0][1] = c.RawTime
	rows[0][2] = c.Source
	rows[0][3] = c.Rating
	rows[0][6] = c.TicketPrice
	rows[0][7] = osc8(c.ReadMoreURL, "Link")

	return rows
}

// osc8 wraps text in an OSC 8 hyperlink escape so the terminal holds the full
// URL behind short display text. Terminals without OSC 8 support just print the
// text. It stays in the last column, whose width tabwriter never measures — the
// escape bytes would otherwise inflate the computed column width.
func osc8(url, text string) string {
	if url == "" {
		return ""
	}
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}
