package main

import (
	"fmt"
	"html/template"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ninkyclaw/pkg/model"
)

// htmlTmpl renders the same columns as the terminal table, with the title
// carrying the read-more link instead of a separate column and the date cell
// leading with an add-to-calendar link.
var htmlTmpl = template.Must(template.New("concerts").Funcs(template.FuncMap{
	"join": strings.Join,
	"past": isPast,
	"gcal": gcalURL,
}).Parse(`<!DOCTYPE html>
<html lang="en">
<meta charset="utf-8">
<meta name="robots" content="noindex">
<title>Concerts</title>
<style>
body { font: 14px/1.4 system-ui, sans-serif; margin: 2rem; }
table { border-collapse: collapse; }
th, td { padding: 0.3rem 0.6rem; text-align: left; vertical-align: top; }
th { border-bottom: 2px solid #888; }
tr:nth-child(even) { background: #f3f3f3; }
tr.past { background: #e0e0e0; color: #777; }
tr.past a { color: #777; }
td:nth-child(-n+2) { white-space: nowrap; }
/* Some sites write a sentence where a price belongs; wrap it instead of
   stretching the table. */
td:last-child { max-width: 20ch; }
</style>
<table>
<tr><th>Date<th>Time<th>Source<th>Rating<th>Keywords<th>Title<th>Ticket
{{range .}}<tr{{if past .Date}} class="past"{{end}}>
<td><a href="{{gcal .}}" target="_blank" rel="noopener" title="Add to Google Calendar">&#128197;</a> {{.Date.Format "Jan 2"}}
<td>{{.RawTime}}
<td>{{.Source}}
<td>{{.Rating}}
<td>{{join .MatchedKeywords ", "}}
<td>{{if .ReadMoreURL}}<a href="{{.ReadMoreURL}}">{{.Title}}</a>{{else}}{{.Title}}{{end}}
<td>{{.TicketPrice}}
{{end}}</table>
`))

// clockRe finds the "19:00" clock times inside a RawTime such as "19:00",
// "19:00 - 22:00" or muba's en-dashed "18:00 – 19:30".
var clockRe = regexp.MustCompile(`(\d{1,2}):(\d{2})`)

// gcalURL builds a Google Calendar "create event" link prefilled from the
// concert. Times are written as naive local clock times and left to ctz, so no
// tzdata lookup is needed; every venue in the list is in Tallinn.
func gcalURL(c model.Concert) string {
	q := url.Values{
		"action":   {"TEMPLATE"},
		"text":     {c.Title},
		"ctz":      {"Europe/Tallinn"},
		"location": {c.Venue},
		"details":  {c.ReadMoreURL},
		"dates":    {eventDates(c)},
	}
	return "https://calendar.google.com/calendar/render?" + q.Encode()
}

// eventDates renders the dates parameter: a clock-time range when RawTime gives
// one, otherwise an all-day event, whose end date Google treats as exclusive.
func eventDates(c model.Concert) string {
	clocks := clockRe.FindAllStringSubmatch(c.RawTime, -1)
	if len(clocks) == 0 {
		return fmt.Sprintf("%s/%s", c.Date.Format("20060102"), c.Date.AddDate(0, 0, 1).Format("20060102"))
	}

	// UTC is just a carrier for the arithmetic here — the zone never gets printed.
	day := time.Date(c.Date.Year(), c.Date.Month(), c.Date.Day(), 0, 0, 0, 0, time.UTC)
	start := day.Add(clockOffset(clocks[0]))
	end := start.Add(2 * time.Hour)
	if len(clocks) > 1 {
		// ponytail: a second clock is assumed to be on the same day, so eccm's rare
		// multi-day "15:51 - 25.10.2026 23:51" collapses to one long evening.
		if stated := day.Add(clockOffset(clocks[1])); stated.After(start) {
			end = stated
		}
	}
	return fmt.Sprintf("%s/%s", start.Format("20060102T150405"), end.Format("20060102T150405"))
}

// clockOffset turns a ["19:00" "19" "00"] match into a duration since midnight.
func clockOffset(m []string) time.Duration {
	// The regexp already proved both groups are digits, so the errors can't fire.
	h, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	return time.Duration(h)*time.Hour + time.Duration(min)*time.Minute
}

// isPast reports whether t falls before today in t's own location — a concert
// earlier today is still upcoming, so the whole day counts as not past.
func isPast(t time.Time) bool {
	now := time.Now().In(t.Location())
	return t.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, t.Location()))
}

// writeHTML renders the concerts as a standalone page at path.
func writeHTML(path string, concerts []model.Concert) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := htmlTmpl.Execute(f, concerts); err != nil {
		return err
	}
	return f.Close()
}
