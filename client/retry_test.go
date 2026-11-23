package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mitchellh/go-homedir"
)

// newTestClient starts a fake API that answers with the given statuses in
// order (the last one repeats) and returns the provider's HTTP client for it
// and a pointer to the number of API calls.
func newTestClient(t *testing.T, statuses ...int) (*http.Client, string, *int) {
	retryBackoff = func(int) time.Duration { return 0 }
	// Keep the token cache separate from the real one; go-homedir caches the
	// home dir, so disable that for HOME to take effect.
	homedir.DisableCache = true
	t.Setenv("HOME", t.TempDir())

	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"x","token_type":"bearer","expires_in":3600}`))
			return
		}
		w.WriteHeader(statuses[min(hits, len(statuses)-1)])
		hits++
	}))
	t.Cleanup(srv.Close)

	secret, pass := "s", "p"
	api, err := NewClient(srv.URL, srv.URL+"/token", "id", &secret, "u", &pass)
	if err != nil {
		t.Fatal(err)
	}
	return api.GetConfig().HTTPClient, srv.URL + "/v1/compute/instances", &hits
}

func TestRetry_Retries429(t *testing.T) {
	c, url, hits := newTestClient(t, 429, 429, 200)
	resp, err := c.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || *hits != 3 {
		t.Fatalf("want 200 after 3 calls, got %d after %d", resp.StatusCode, *hits)
	}
}

// Repeating a POST that failed with 5xx could order a duplicate paid resource.
func TestRetry_PostNotRetriedOn5xx(t *testing.T) {
	c, url, hits := newTestClient(t, 500, 201)
	resp, err := c.Post(url, "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 500 || *hits != 1 {
		t.Fatalf("want 500 after 1 call, got %d after %d", resp.StatusCode, *hits)
	}
}
