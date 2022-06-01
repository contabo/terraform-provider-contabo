package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hprose/hprose-go"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/oauth2"
)

func BearerHttpClient(
	authUrl string,
	clientId string,
	clientSecret string,
	username string,
	password string,
) (*http.Client, error) {
	ctx := context.Background()
	configuration := &oauth2.Config{
		ClientID: clientId,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL: authUrl,
		},
	}

	// check if token has been cached
	token, err := RestoreTokenFromCache()

	if err != nil {
		return nil, err
	}

	if token == nil {
		token, err = configuration.PasswordCredentialsToken(ctx, username, password)
		if err != nil {
			return nil, fmt.Errorf("error while getting access token: %s", err)
		}
	}

	err = cacheToken(token)

	if err != nil {
		return nil, err
	}

	return configuration.Client(ctx, token), nil
}

func cacheToken(token *oauth2.Token) error {
	serializedToken, err := hprose.Serialize(token, true)
	if err != nil {
		return fmt.Errorf("could not serialize token due to erros %v", err)
	}

	tokenCacheFileName, err := getCacheFile()
	if err != nil {
		return err
	}

	tokenCacheFile, err := os.Create(*tokenCacheFileName)
	if err != nil {
		return fmt.Errorf("could not open cache file %v due to errors %v", tokenCacheFile, err)
	}

	_, err = tokenCacheFile.Write(serializedToken)
	tokenCacheFile.Close()
	if err != nil {
		return fmt.Errorf("could not write to cache file %v due to errors %v", tokenCacheFile, err)
	}

	return nil
}

func RestoreTokenFromCache() (*oauth2.Token, error) {
	tokenCacheFileName, err := getCacheFile()
	if err != nil {
		return nil, err
	}

	serializedToken, err := ioutil.ReadFile(*tokenCacheFileName)
	if err != nil {
		return nil, nil
	}

	var token oauth2.Token
	err = hprose.Unserialize(serializedToken, &token, true)
	if err != nil {
		return nil, fmt.Errorf("could not deserialize token due to erros %v", err)
	}
	// check token expiry, take refresh token expiry date if present otherwise fall back to access token expiry
	accessTokenExpired := token.Expiry.Before(time.Now())
	jwt := strings.Split(token.RefreshToken, ".")[1]
	jwtPayloadAsBytes, _ := base64.RawStdEncoding.DecodeString(jwt)
	type JwtPayload struct {
		Exp int64 `json:"exp"`
	}
	var jwtPayload JwtPayload
	json.Unmarshal(jwtPayloadAsBytes, &jwtPayload)
	refreshTokenExpiryDate := time.Unix(jwtPayload.Exp, 0)
	refreshTokenExpired := refreshTokenExpiryDate.Before(time.Now())

	if accessTokenExpired && refreshTokenExpired {
		return nil, nil
	}
	return &token, nil
}

func getCacheFile() (*string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, fmt.Errorf("could not determine home dir: %v", err)
	}
	tokenCacheDirName := home + "/.cache/contabo/terraform"
	err = os.MkdirAll(tokenCacheDirName, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("could not ensure cache folder: %v", err)
	}
	tokenCacheFileName := home + "/.cache/contabo/terraform/token"
	return &tokenCacheFileName, nil
}