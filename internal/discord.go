package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

const DiscordApiURI = "https://discord.com/api/v10"
const TokenSize = 128

type DiscordClient struct {
	ClientID     string
	ClientSecret string
	Oauth2URI    string
}

// FetchUsername Retrieve discord username from oauth2 code
func (d *DiscordClient) FetchUsername(code string) (string, error) {
	client := http.Client{}

	var payload map[string]interface{}
	var err error
	var buf []byte
	var response *http.Response

	form := url.Values{
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"client_id":     {d.ClientID},
		"client_secret": {d.ClientSecret},
		"redirect_uri":  {d.Oauth2URI},
	}

	response, err = client.PostForm(fmt.Sprintf("%s/oauth2/token", DiscordApiURI), form)
	if err != nil {
		log.Println(err)
		return "", err
	}

	buf, err = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		log.Println(err)
		return "", err
	}

	if response.StatusCode != http.StatusOK {
		log.Println("bad response status code from oauth2")
		log.Println(string(buf))
		return "", errors.New("bad response from oauth2")
	}

	err = json.Unmarshal(buf, &payload)
	if err != nil {
		log.Println(err)
		return "", err
	}

	if payload["access_token"] == nil {
		log.Println("missing 'access_token' field from discord api response")
		log.Println(string(buf))
		return "", err
	}

	token := payload["access_token"].(string)
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/users/@me", DiscordApiURI), nil)
	if err != nil {
		log.Println(err)
		return "", err
	}

	request.Header = http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer %s", token)},
	}

	response, err = client.Do(request)
	if err != nil {
		log.Println(err)
		return "", err
	}

	buf, err = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		log.Println(err)
		return "", err
	}

	err = json.Unmarshal(buf, &payload)
	if err != nil {
		log.Println(err)
		return "", err
	}

	if payload["username"] == nil {
		log.Println("missing 'username' field from discord api response")
		log.Println(string(buf))
		return "", err
	}

	return payload["username"].(string), nil
}
