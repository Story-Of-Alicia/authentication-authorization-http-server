package oauth2

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"soaauth/internal/config"
	"soaauth/internal/types"
)

type discordResponse struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	Token    string `json:"access_token"`
	Username string `json:"username"`
}

var cfg *config.Config = config.GetConfigInstance()

func DiscordGetUsername(code string) (string, *types.APIError) {
	client := http.Client{}
	var respBody discordResponse
	token, error := getToken(client, code)

	if error != nil {
		return "", error
	}

	request, err := http.NewRequest(http.MethodGet, cfg.DiscordConfig.DiscordApiUri+"/users/@me/", nil)
	if err != nil {
		return "", &types.APIError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}
	}

	request.Header = http.Header{
		"Authorization": []string{"Bearer " + token},
	}

	response, err := client.Do(request)

	if err != nil {
		return "", &types.APIError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}
	}

	err = decodeRespBody(response.Body, &respBody)

	if err != nil {
		return "", &types.APIError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}
	}

	if response.StatusCode != http.StatusOK {
		return "", &types.APIError{
			Code:    response.StatusCode,
			Message: respBody.Message,
		}
	}

	return respBody.Username, nil
}

func getToken(client http.Client, code string) (string, *types.APIError) {
	var respBody discordResponse

	form := url.Values{
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"client_id":     {cfg.DiscordConfig.ClientID},
		"client_secret": {cfg.DiscordConfig.ClientSecret},
		"redirect_uri":  {cfg.DiscordConfig.RedirectUri},
	}

	response, err := client.PostForm(cfg.DiscordConfig.DiscordApiUri+"/oauth2/token/", form)
	if err != nil {
		return "", &types.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		}
	}

	err = decodeRespBody(response.Body, &respBody)

	if response.StatusCode != http.StatusOK {
		return "", &types.APIError{
			Code:    response.StatusCode,
			Message: respBody.Message,
		}
	}

	return respBody.Token, nil
}

func decodeRespBody(body io.ReadCloser, respBody *discordResponse) error {
	defer body.Close()

	buff, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(buff, &respBody); err != nil {
		return err
	}

	return nil
}
