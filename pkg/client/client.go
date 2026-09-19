package client

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout = 15 * time.Second
	maxRetries     = 2
	maxBackoff     = 30 * time.Second
)

// Client is a generic HTTP client for fetching web pages.
type Client struct {
	HTTPClient *http.Client
}

// NewClient initializes a Client with a sensible default timeout.
func NewClient() *Client {
	return &Client{
		HTTPClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// Fetch performs a GET request to the given URL and returns the response body.
// The caller is responsible for closing the returned ReadCloser.
func (c *Client) Fetch(url string) (io.ReadCloser, error) {
	return c.do("GET", url, "")
}

// Post performs an application/x-www-form-urlencoded POST request and returns
// the response body. The caller is responsible for closing the returned ReadCloser.
func (c *Client) Post(url, formData string) (io.ReadCloser, error) {
	return c.do("POST", url, formData)
}

// do sends the request, waiting and trying again while the server answers 429.
// The request is rebuilt on every attempt because a retry re-reads the body.
func (c *Client) do(method, url, formData string) (io.ReadCloser, error) {
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequest(method, url, strings.NewReader(formData))
		if err != nil {
			return nil, fmt.Errorf("%s %s: %w", method, url, err)
		}
		if formData != "" {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("%s %s: %w", method, url, err)
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxRetries {
			wait := retryAfter(resp.Header.Get("Retry-After"), attempt)
			resp.Body.Close()
			time.Sleep(wait)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("%s %s: unexpected status %d", method, url, resp.StatusCode)
		}

		return resp.Body, nil
	}
}

// retryAfter reads the Retry-After header, which is either a number of seconds
// or an HTTP date. Servers often send neither, so an unreadable value falls
// back to doubling seconds, and an absurd one is capped rather than obeyed.
func retryAfter(header string, attempt int) time.Duration {
	wait := time.Duration(1<<attempt) * time.Second

	if secs, err := strconv.Atoi(strings.TrimSpace(header)); err == nil {
		wait = time.Duration(secs) * time.Second
	} else if date, err := http.ParseTime(header); err == nil {
		wait = time.Until(date)
	}

	if wait > maxBackoff {
		return maxBackoff
	}
	if wait < 0 {
		return 0
	}
	return wait
}
