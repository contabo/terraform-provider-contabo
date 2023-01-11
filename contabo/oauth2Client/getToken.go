package oauth2Client

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

func GetJwtToken(
	authUrl string,
	clientId string,
	clientSecret *string,
	username string,
	password *string,
) {

	strings.NewReader(`asd {{ apiUrl }}`)

	urlEncodedUsername := url.QueryEscape(username)
	urlEncodedPassword := url.QueryEscape(*password)

	payload := strings.NewReader("client_id=" + clientId + "&client_secret=" + *clientSecret + "&username=" + urlEncodedUsername + "&password=" + urlEncodedPassword + "&grant_type=password")

	client := &http.Client{}
	req, err := http.NewRequest("POST", authUrl, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}
