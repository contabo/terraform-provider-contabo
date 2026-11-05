package client

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func testJWT(claims map[string]interface{}) string {
	payload, _ := json.Marshal(claims)
	return "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

func tokenServer(t *testing.T, accessTTL int64, refreshTTL int64, logins *int32, refreshes *int32) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch r.Form.Get("grant_type") {
		case "password":
			atomic.AddInt32(logins, 1)
		case "refresh_token":
			atomic.AddInt32(refreshes, 1)
		}
		now := time.Now().Unix()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  testJWT(map[string]interface{}{"sub": "user-" + r.Form.Get("username"), "exp": now + accessTTL}),
			"refresh_token": testJWT(map[string]interface{}{"exp": now + refreshTTL}),
			"expires_in":    accessTTL,
			"token_type":    "Bearer",
		})
	}))
	t.Cleanup(server.Close)
	return server
}

func TestTokenCacheIsPerCredentials(t *testing.T) {
	t.Setenv("CNTB_TOKEN_CACHE_DIR", t.TempDir())
	var logins, refreshes int32
	server := tokenServer(t, 300, 1800, &logins, &refreshes)

	first, err := BearerHttpClient(server.URL, "client", "secret", "alice", "pw")
	if err != nil {
		t.Fatal(err)
	}
	second, err := BearerHttpClient(server.URL, "client", "secret", "bob", "pw")
	if err != nil {
		t.Fatal(err)
	}
	if sub, _ := AccessTokenSubject(first); sub != "user-alice" {
		t.Fatalf("first client uses token of %q", sub)
	}
	if sub, _ := AccessTokenSubject(second); sub != "user-bob" {
		t.Fatalf("second client uses token of %q", sub)
	}

	// A third client for alice reuses the cached token.
	if _, err := BearerHttpClient(server.URL, "client", "secret", "alice", "pw"); err != nil {
		t.Fatal(err)
	}
	if logins != 2 {
		t.Fatalf("expected 2 logins, got %d", logins)
	}
}

func TestTokenRefreshAndRelogin(t *testing.T) {
	t.Setenv("CNTB_TOKEN_CACHE_DIR", t.TempDir())
	var logins, refreshes int32
	// Access tokens expire within the safety margin, so every use refreshes.
	server := tokenServer(t, 1, 1800, &logins, &refreshes)
	httpClient, err := BearerHttpClient(server.URL, "client", "secret", "alice", "pw")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AccessTokenSubject(httpClient); err != nil {
		t.Fatal(err)
	}
	if refreshes == 0 {
		t.Fatalf("expired access token was not refreshed")
	}

	// Refresh tokens that are expired too lead to a new login.
	var logins2, refreshes2 int32
	server2 := tokenServer(t, 1, 1, &logins2, &refreshes2)
	httpClient2, err := BearerHttpClient(server2.URL, "client", "secret", "carol", "pw")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AccessTokenSubject(httpClient2); err != nil {
		t.Fatal(err)
	}
	if logins2 < 2 || refreshes2 != 0 {
		t.Fatalf("expected re-login without refresh, got %d logins and %d refreshes", logins2, refreshes2)
	}
}
