package model

import "time"

// Concert represents an extracted concert event from the EMTA calendar.
type Concert struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Date        time.Time `json:"date"`
	RawTime     string    `json:"raw_time"`      // e.g. "kell 19:00"
	Venue       string    `json:"venue"`         // e.g. "Suur saal"
	Description string    `json:"description"`
	ReadMoreURL         string    `json:"read_more_url"` // Link to detailed event page
	ImageURL            string    `json:"image_url,omitempty"`
	ExtendedDescription string    `json:"extended_description,omitempty"`
	TicketPrice         string    `json:"ticket_price"` // e.g., "Free", "Paid"
	TicketURL           string    `json:"ticket_url,omitempty"`
	Rating              string    `json:"rating,omitempty"`
	Source              string    `json:"source,omitempty"`
}
