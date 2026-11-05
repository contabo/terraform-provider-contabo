package client

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hprose/hprose-go"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/oauth2"
)

// tokenExpiryMargin makes cached access tokens count as expired a little
// before they really expire, so a request never starts with a token that
// expires in flight.
const tokenExpiryMargin = 30 * time.Second

// BearerHttpClient returns an HTTP client that adds a bearer token to every
// request. The token is obtained with the OAuth2 password grant, cached on
// disk per token URL, client id and user, refreshed with the refresh token,
// and obtained again with the password grant when the refresh token has
// expired too.
func BearerHttpClient(
	authUrl string,
	clientId string,
	clientSecret string,
	username string,
	password string,
) (*http.Client, error) {
	source, err := newPasswordTokenSource(authUrl, clientId, clientSecret, username, password)
	if err != nil {
		return nil, err
	}
	return oauth2.NewClient(context.Background(), source), nil
}

// AccessTokenSubject returns the "sub" claim of the current access token,
// which is the Contabo user id of the API user.
func AccessTokenSubject(httpClient *http.Client) (string, error) {
	transport, ok := httpClient.Transport.(*oauth2.Transport)
	if !ok {
		return "", fmt.Errorf("http client does not use an oauth2 transport")
	}
	token, err := transport.Source.Token()
	if err != nil {
		return "", err
	}
	claims, err := jwtClaims(token.AccessToken)
	if err != nil {
		return "", err
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", fmt.Errorf("access token has no subject claim")
	}
	return sub, nil
}

type passwordTokenSource struct {
	mu        sync.Mutex
	config    *oauth2.Config
	username  string
	password  string
	cacheFile string
	token     *oauth2.Token
}

func newPasswordTokenSource(
	authUrl string,
	clientId string,
	clientSecret string,
	username string,
	password string,
) (*passwordTokenSource, error) {
	cacheFile, err := getCacheFile(authUrl, clientId, username)
	if err != nil {
		return nil, err
	}
	source := &passwordTokenSource{
		config: &oauth2.Config{
			ClientID:     clientId,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				TokenURL:  authUrl,
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		username:  username,
		password:  password,
		cacheFile: cacheFile,
	}
	// A broken or unreadable cache only costs one extra login.
	source.token = restoreTokenFromCache(cacheFile)
	if _, err := source.Token(); err != nil {
		return nil, err
	}
	return source, nil
}

func (s *passwordTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != nil && s.token.Expiry.After(time.Now().Add(tokenExpiryMargin)) {
		return s.token, nil
	}

	ctx := context.Background()
	var token *oauth2.Token
	var err error
	if s.token != nil && s.token.RefreshToken != "" && !refreshTokenExpired(s.token) {
		token, err = s.config.TokenSource(ctx, &oauth2.Token{RefreshToken: s.token.RefreshToken}).Token()
	}
	if token == nil || err != nil {
		token, err = s.config.PasswordCredentialsToken(ctx, s.username, s.password)
		if err != nil {
			return nil, fmt.Errorf("error while getting access token: %s", err)
		}
	}

	s.token = token
	// Caching is an optimization; a read-only home directory must not fail
	// the provider.
	_ = cacheToken(s.cacheFile, token)
	return token, nil
}

func cacheToken(cacheFile string, token *oauth2.Token) error {
	serializedToken, err := hprose.Serialize(token, true)
	if err != nil {
		return fmt.Errorf("could not serialize token: %v", err)
	}
	if err := os.WriteFile(cacheFile, serializedToken, 0o600); err != nil {
		return fmt.Errorf("could not write token cache file %s: %v", cacheFile, err)
	}
	return nil
}

func restoreTokenFromCache(cacheFile string) *oauth2.Token {
	serializedToken, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil
	}
	var token oauth2.Token
	if err := hprose.Unserialize(serializedToken, &token, true); err != nil {
		return nil
	}
	if token.AccessToken == "" {
		return nil
	}
	if token.Expiry.Before(time.Now()) && refreshTokenExpired(&token) {
		return nil
	}
	return &token
}

// refreshTokenExpired reads the expiry from the refresh token, which
// Keycloak issues as a JWT. Tokens that cannot be decoded count as expired.
func refreshTokenExpired(token *oauth2.Token) bool {
	claims, err := jwtClaims(token.RefreshToken)
	if err != nil {
		return true
	}
	exp, ok := claims["exp"].(float64)
	if !ok {
		return true
	}
	return time.Unix(int64(exp), 0).Before(time.Now().Add(tokenExpiryMargin))
}

func jwtClaims(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("token is not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return nil, fmt.Errorf("could not decode JWT payload: %v", err)
	}
	claims := map[string]interface{}{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("could not parse JWT payload: %v", err)
	}
	return claims, nil
}

// getCacheFile returns the token cache file for one set of credentials, so
// that providers configured for different accounts or API endpoints never
// share a token. CNTB_TOKEN_CACHE_DIR overrides the cache directory.
func getCacheFile(authUrl string, clientId string, username string) (string, error) {
	dir := os.Getenv("CNTB_TOKEN_CACHE_DIR")
	if dir == "" {
		home, err := homedir.Dir()
		if err != nil {
			return "", fmt.Errorf("could not determine home dir: %v", err)
		}
		dir = filepath.Join(home, ".cache", "contabo", "terraform")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("could not ensure cache folder: %v", err)
	}
	sum := sha256.Sum256([]byte(authUrl + "\x00" + clientId + "\x00" + username))
	return filepath.Join(dir, "token-"+hex.EncodeToString(sum[:8])), nil
}
