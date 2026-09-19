# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Go CLI that scrapes concert calendars from six Estonian/Tallinn venue sites, rates each event against a keyword rules file, and prints a chronological table.

## Commands

```bash
go run ./cmd/ninkyclaw concerts                                    # current month, all sources
go run ./cmd/ninkyclaw concerts -year 2026 -month 6 -source muba   # one source, one month
go run ./cmd/ninkyclaw concerts -rules other.csv                   # ratings from a different rules file
go run ./cmd/ninkyclaw concerts -months 2 -html site/index.html    # what the daily Pages workflow runs
go run ./cmd/ninkyclaw concerts -html out.html                     # HTML page instead of the table

gofmt -l .                                                # must print nothing
go test ./...                                             # all parser tests (offline)
go test ./pkg/scrape/muba                                 # one package
go test ./pkg/scrape/emta -run TestParseCalendar -v        # one test
```

Sources: `all` (default), `emta`, `concert`, `filharmoonia`, `muba`, `eccm`, `phillyjoes`.

## Architecture

Each site gets its own package under `pkg/scrape/<site>/parser.go` implementing `scrape.Scraper` (`Scrape(year, month int) ([]model.Concert, error)`). `cmd/ninkyclaw/main.go` holds the scrapers in one `name`/`scrape.Scraper` slice and loops over it once per month in the `-months` window, collecting `[]model.Concert`, then rates, sorts, prints. A scraper failure is logged and skipped, not fatal — partial results are expected output.

The key split inside every scraper package:

- **`Scrape`** does I/O: builds URLs/POST bodies, fetches, follows pagination, enriches with detail pages.
- **`Parse(io.Reader)`** is pure and takes a reader — this is what tests exercise against fixture HTML, so no test hits the network.

Keep that boundary when adding a source, otherwise the package becomes untestable.

Every site delivers its month differently, and this is where the real work lives:

| Source | Mechanism |
|---|---|
| `emta` | Page ignores URL params; POST to the WordPress `admin-ajax.php` endpoint the site's JS calls |
| `phillyjoes` | Squarespace JSON API (`GetItemsByMonth`), epoch-ms dates |
| `muba` | `?date_from=&date_to=` query + follow `nextURL` pagination in a loop |
| `filharmoonia` | Single full listing; filter by year/month in Go, then fetch each detail page — the listing JSON leaves `description` empty for about a third of them |
| `eccm`, `concert` | HTML calendar + per-event detail page fetch |

`pkg/client` is the shared HTTP wrapper (15s timeout, `Fetch`/`Post`, non-200 is an error). Use it rather than `net/http` directly.

`model.Concert.ID`: sites with a native numeric event ID use it (`emta`, `concert`); the rest hash a stable string with `fnv.New32a`. IDs are not globally unique across sources — `Source` disambiguates.

## Rating

`pkg/rating` matches lowercase keywords from a CSV (`keyword,rating`) against `Title` + `Description` + `ExtendedDescription` (several sources ship an empty description, so the title carries the composer), taking the highest match; unmatched is `Low`, the lowest rating. `rules.csv` sits at the repo root and is the default `-rules` value; if the file is missing the run continues and everything gets `Low`.

## Tests

Fixtures live in `testdata/<source>_{calendar,detail}_fixture.{html,json}` — every package reads them from `filepath.Join("..", "..", "..", "testdata", ...)`. The exception is `muba`, whose test uses an inline HTML string literal.

Fixtures must be committed: `go test ./...` is expected to pass on a fresh clone, so verify with a real clone rather than trusting a local run, which happily reads untracked files.

`scratch/` and `scratch_*` at the repo root are gitignored raw page dumps, kept by hand for refreshing fixtures when a site's markup changes. Nothing in the build may read them.

## Publishing

`.github/workflows/pages.yml` runs daily at 05:00 UTC (and on demand via
`workflow_dispatch`), scrapes the current plus next month into `site/index.html`, and
deploys it to GitHub Pages. It greps the page for a `<td>` before uploading, because a
failed scraper is only logged — without that check an empty page would overwrite a good
one. GitHub disables scheduled workflows after 60 days without repo activity.
