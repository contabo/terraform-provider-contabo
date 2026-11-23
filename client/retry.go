package client

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const maxAttempts = 4

// retryBackoff returns the wait before the given retry attempt (1-based).
// It is a variable so tests can shorten it.
var retryBackoff = func(attempt int) time.Duration {
	return time.Duration(attempt*2) * time.Second
}

// retryRoundTripper retries transient Contabo API failures.
//
// Idempotent requests (GET, HEAD, OPTIONS, PUT, DELETE) are retried on
// network errors, 429 and 5xx gateway/server errors. Non-idempotent requests
// (POST, PATCH) are retried only on 429: any other failure may mean the
// request was already applied (e.g. an instance was ordered), and repeating
// it could create a duplicate paid resource.
type retryRoundTripper struct {
	base http.RoundTripper
}

func (r *retryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := r.base
	if base == nil {
		base = http.DefaultTransport
	}

	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, err
		}
	}

	idempotent := isIdempotent(req.Method)

	for attempt := 1; ; attempt++ {
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err := base.RoundTrip(req)

		retry := false
		if err != nil {
			retry = idempotent
		} else if resp.StatusCode == http.StatusTooManyRequests {
			retry = true
		} else if idempotent && isRetryableServerError(resp.StatusCode) {
			retry = true
		}

		if !retry || attempt >= maxAttempts {
			return resp, err
		}

		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		wait := retryBackoff(attempt)
		log.Printf("[WARN] Contabo API %s %s: transient error (status: %s, err: %v), retrying in %v (attempt %d/%d)",
			req.Method, req.URL.Path, statusCode(resp), err, wait, attempt+1, maxAttempts)

		timer := time.NewTimer(wait)
		select {
		case <-req.Context().Done():
			timer.Stop()
			return nil, req.Context().Err()
		case <-timer.C:
		}
	}
}

func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete:
		return true
	}
	return false
}

func isRetryableServerError(code int) bool {
	switch code {
	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

func statusCode(resp *http.Response) string {
	if resp == nil {
		return "none"
	}
	return fmt.Sprintf("%d", resp.StatusCode)
}
