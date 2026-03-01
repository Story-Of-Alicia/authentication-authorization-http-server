package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const DiscordApiURI = "https://discord.com/api/v10"
const TokenSize = 128

type DiscordClient struct {
	ClientID     string
	ClientSecret string
	Oauth2URI    string
	Ctx          context.Context
}

// FetchUserID Retrieve discord id from oauth2 code
func (d *DiscordClient) FetchUserID(code string) (string, error) {
	/* if context is canceled, return error */
	select {
	default:
	case <-d.Ctx.Done():
		return "", d.Ctx.Err()
	}

	ctx, cancel := context.WithTimeout(d.Ctx, 20*time.Second)
	defer cancel()

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

	/* TODO: use context with this request */
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
	request, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/users/@me", DiscordApiURI), nil)
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

	if payload["id"] == nil {
		log.Println("missing 'id' field from discord api response")
		log.Println(string(buf))
		return "", err
	}

	return payload["id"].(string), nil
}
