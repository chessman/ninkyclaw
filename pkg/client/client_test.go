package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchRetriesOn429(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		io.WriteString(w, "ok")
	}))
	defer srv.Close()

	body, err := NewClient().Fetch(srv.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	defer body.Close()

	got, _ := io.ReadAll(body)
	if string(got) != "ok" {
		t.Errorf("body = %q, want %q", got, "ok")
	}
	if calls != 2 {
		t.Errorf("server saw %d calls, want 2", calls)
	}
}

func TestFetchGivesUpOn429(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	if _, err := NewClient().Fetch(srv.URL); err == nil {
		t.Fatal("expected an error after the retries ran out")
	}
	if calls != maxRetries+1 {
		t.Errorf("server saw %d calls, want %d", calls, maxRetries+1)
	}
}

func TestRetryAfter(t *testing.T) {
	for _, tt := range []struct {
		header  string
		attempt int
		want    time.Duration
	}{
		{"5", 0, 5 * time.Second},
		{"", 0, time.Second},
		{"", 2, 4 * time.Second},
		{"nonsense", 1, 2 * time.Second},
		{"9999", 0, maxBackoff},
		{time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat), 0, 0},
	} {
		if got := retryAfter(tt.header, tt.attempt); got != tt.want {
			t.Errorf("retryAfter(%q, %d) = %v, want %v", tt.header, tt.attempt, got, tt.want)
		}
	}
}
