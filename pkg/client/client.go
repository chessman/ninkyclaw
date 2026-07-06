package client

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultTimeout = 15 * time.Second

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
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: unexpected status %d", url, resp.StatusCode)
	}

	return resp.Body, nil
}

// Post performs an application/x-www-form-urlencoded POST request and returns
// the response body. The caller is responsible for closing the returned ReadCloser.
func (c *Client) Post(url, formData string) (io.ReadCloser, error) {
	resp, err := c.HTTPClient.Post(url, "application/x-www-form-urlencoded", strings.NewReader(formData))
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("POST %s: unexpected status %d", url, resp.StatusCode)
	}

	return resp.Body, nil
}

