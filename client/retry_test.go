package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mitchellh/go-homedir"
)

func noBackoff(t *testing.T) {
	orig := retryBackoff
	retryBackoff = func(int) time.Duration { return 0 }
	t.Cleanup(func() { retryBackoff = orig })
}

// fakeAPI serves a token endpoint and answers API calls with the given
// statuses in order (the last one repeats). It counts API calls.
func fakeAPI(t *testing.T, statuses ...int) (*httptest.Server, *int32) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"x","token_type":"bearer","expires_in":3600}`))
			return
		}
		body, _ := io.ReadAll(r.Body)
		n := int(atomic.AddInt32(&hits, 1))
		if n > len(statuses) {
			n = len(statuses)
		}
		w.WriteHeader(statuses[n-1])
		w.Write([]byte(`{"statusCode":` + http.StatusText(statuses[n-1]) + `,"echo":"` + string(body) + `"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// newTestHTTPClient builds the HTTP client exactly as the provider does.
func newTestHTTPClient(t *testing.T, srv *httptest.Server) *http.Client {
	// Keep the token cache away from the real one and separate per test.
	// go-homedir caches the home dir, so disable that for HOME to take effect.
	homedir.DisableCache = true
	t.Cleanup(func() { homedir.DisableCache = false })
	t.Setenv("HOME", t.TempDir())
	secret, pass := "s", "p"
	api, err := NewClient(srv.URL, srv.URL+"/token", "id", &secret, "u", &pass)
	if err != nil {
		t.Fatal(err)
	}
	return api.GetConfig().HTTPClient
}

func TestRetry_ProviderClientRetries429(t *testing.T) {
	noBackoff(t)
	srv, hits := fakeAPI(t, 429, 429, 200)
	resp, err := newTestHTTPClient(t, srv).Get(srv.URL + "/v1/compute/instances")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || *hits != 3 {
		t.Fatalf("want 200 after 3 calls, got %d after %d", resp.StatusCode, *hits)
	}
}

func TestRetry_GetRetries5xx(t *testing.T) {
	noBackoff(t)
	srv, hits := fakeAPI(t, 502, 503, 200)
	resp, err := newTestHTTPClient(t, srv).Get(srv.URL + "/v1/compute/instances")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || *hits != 3 {
		t.Fatalf("want 200 after 3 calls, got %d after %d", resp.StatusCode, *hits)
	}
}

func TestRetry_GivesUpWithReadableBody(t *testing.T) {
	noBackoff(t)
	srv, hits := fakeAPI(t, 500)
	resp, err := newTestHTTPClient(t, srv).Get(srv.URL + "/v1/compute/instances")
	if err != nil {
		t.Fatal(err)
	}
	if *hits != maxAttempts {
		t.Fatalf("want %d calls, got %d", maxAttempts, *hits)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil || len(body) == 0 {
		t.Fatalf("final error body must stay readable for HandleResponseErrors, got %q, err %v", body, err)
	}
}

func TestRetry_PostNotRetriedOn5xx(t *testing.T) {
	noBackoff(t)
	srv, hits := fakeAPI(t, 500, 201)
	resp, err := newTestHTTPClient(t, srv).Post(srv.URL+"/v1/compute/instances", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 500 || *hits != 1 {
		t.Fatalf("POST must not be repeated on 500 (could order a duplicate), got %d after %d calls", resp.StatusCode, *hits)
	}
}

func TestRetry_PostRetriedOn429WithBody(t *testing.T) {
	noBackoff(t)
	srv, hits := fakeAPI(t, 429, 201)
	resp, err := newTestHTTPClient(t, srv).Post(srv.URL+"/v1/compute/instances", "application/json", strings.NewReader(`payload`))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 201 || *hits != 2 || !strings.Contains(string(body), "payload") {
		t.Fatalf("want 201 after 2 calls with body resent, got %d after %d, body %q", resp.StatusCode, *hits, body)
	}
}
