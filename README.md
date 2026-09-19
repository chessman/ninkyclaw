# EMTA Calendar Scraper in Go

A lightweight, robust command-line scraper and library to extract concert schedules from the **Estonian Academy of Music and Theatre (EMTA)** calendar website.

## Requirements

* Go 1.18 or higher

## Setup

Clone the repository and download dependencies:
```bash
go mod tidy
```

## Running the Scraper

To fetch the concert schedule for the current month and output in JSON (default):
```bash
go run ./cmd/ninkyclaw concerts
```

### Specifying Dates

You can specify a target year and month using the `-year` and `-month` flags:
```bash
go run ./cmd/ninkyclaw concerts -year 2026 -month 6
```

### HTML Output

By default the results print as a table. With `-html` they are written as a standalone
HTML page instead, with each title linking to the event:
```bash
go run ./cmd/ninkyclaw concerts -year 2026 -month 6 -html concerts.html
```

## Running Tests

To run the offline parser unit tests:
```bash
go test ./...
```
