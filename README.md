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
go run ./cmd/ninkyclaw
```

### Specifying Dates

You can specify a target year and month using the `-year` and `-month` flags:
```bash
go run ./cmd/ninkyclaw -year 2026 -month 6
```

### Output Formats

The CLI supports `json`, `csv`, and `table` formatting using the `-format` flag:

**Table Format:**
```bash
go run ./cmd/ninkyclaw -year 2026 -month 6 -format table
```

**CSV Format:**
```bash
go run ./cmd/ninkyclaw -year 2026 -month 6 -format csv
```

## Running Tests

To run the offline parser unit tests:
```bash
go test ./...
```
