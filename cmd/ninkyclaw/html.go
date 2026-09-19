package main

import (
	"html/template"
	"os"
	"strings"
	"time"

	"ninkyclaw/pkg/model"
)

// htmlTmpl renders the same columns as the terminal table, with the title
// carrying the read-more link instead of a separate column.
var htmlTmpl = template.Must(template.New("concerts").Funcs(template.FuncMap{
	"join": strings.Join,
	"past": isPast,
}).Parse(`<!DOCTYPE html>
<html lang="en">
<meta charset="utf-8">
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
</style>
<table>
<tr><th>Date<th>Time<th>Source<th>Rating<th>Keywords<th>Title<th>Ticket
{{range .}}<tr{{if past .Date}} class="past"{{end}}>
<td>{{.Date.Format "2006-01-02"}}
<td>{{.RawTime}}
<td>{{.Source}}
<td>{{.Rating}}
<td>{{join .MatchedKeywords ", "}}
<td>{{if .ReadMoreURL}}<a href="{{.ReadMoreURL}}">{{.Title}}</a>{{else}}{{.Title}}{{end}}
<td>{{.TicketPrice}}
{{end}}</table>
`))

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
