package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

const DiscordApiURI = "https://discord.com/api/v10"
const TokenSize = 128

type server struct {
	db  *sql.DB
	ctx context.Context
	srv *http.Server

	discordSecret string
	discordId     string
	data          string
	addr          string
	oauth2Uri     string
	forwardUri    string
}

func generateToken(size int) string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLKMNOPQRSTVWXYZ0123456789"
	b := make([]byte, size)
	r, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}

	if r != size {
		log.Fatal("failed to generate random token")
	}

	for i := range b {
		b[i] = chars[b[i]%byte(len(chars))]
	}

	return string(b)
}

func (s *server) discordGetUsername(code string) (string, error) {
	client := http.Client{}

	var payload map[string]interface{}
	var err error
	var buf []byte
	var response *http.Response

	form := url.Values{
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"client_id":     {s.discordId},
		"client_secret": {s.discordSecret},
		"redirect_uri":  {s.oauth2Uri},
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

func (s *server) createSession(username string) (string, error) {
	filename := fmt.Sprintf("%s/%s.json", s.data, username)
	var profile map[string]interface{}
	_, err := os.Stat(filename)
	if err == nil {
		/* file exists, read existing contents */
		b, err := os.ReadFile(filename)
		if err != nil {
			log.Println(err)
			return "", err
		}

		err = json.Unmarshal(b, &profile)
		if err != nil {
			log.Println(err)
			return "", err
		}
	} else if os.IsNotExist(err) {
		/* file doesn't exist, sensible defaults */
		profile = map[string]interface{}{
			"characterUid": 0,
			"infractions":  make([]string, 0),
			"name":         username,
			"token":        "dev",
		}
	} else {
		/* just an error */
		log.Println(err)
		return "", err
	}

	token := generateToken(TokenSize)
	profile["token"] = token

	buf, err := json.Marshal(profile)
	if err != nil {
		log.Println(err)
		return "", err
	}

	err = os.WriteFile(filename, buf, 0644)
	if err != nil {
		log.Println(err)
		return "", err
	}

	return token, nil
}

func (s *server) httpHandleCallback(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !req.URL.Query().Has("code") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	code := req.URL.Query().Get("code")

	regex, err := regexp.Compile("^[A-z0-9]*$")
	if err != nil {
		log.Fatal(err)
	}

	if !regex.MatchString(code) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	username, err := s.discordGetUsername(code)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	token, err := s.createSession(username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, fmt.Sprintf("%s/?token=%s&username=%s", s.forwardUri, token, username), http.StatusFound)
}

func (s *server) httpPrepare() {
	s.srv = &http.Server{}
	s.srv.Addr = s.addr

	http.HandleFunc("/callback", s.httpHandleCallback)
}

func (s *server) parseEnv() {
	env := os.Environ()
	for _, e := range env {
		k := strings.Split(e, "=")[0]
		v := strings.Split(e, "=")[1]
		switch {
		case k == "DISCORD_CLIENT_SECRET":
			s.discordSecret = v
		case k == "DISCORD_CLIENT_ID":
			s.discordId = v
		case k == "DATA_DIR":
			s.data = v
		case k == "BIND_ADDR":
			s.addr = v
		case k == "OAUTH2_URI":
			s.oauth2Uri = v
		case k == "FORWARD_URI":
			s.forwardUri = v
		}
	}
}

/* environment parameters:
 * DISCORD_CLIENT_SECRET -> discord application secret
 * DISCORD_CLIENT_ID -> discord application id
 * BIND_ADDR -> http server bind address
 * OAUTH2_URI -> oauth2 redirection uri
 * FORWARD_URI -> where to forward request upon successful authentication
 * DATA_DIR    -> where to store user json files
 */
func main() {
	/* set verbose logging */
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var h = server{}
	h.parseEnv()

	ctx, stop := context.WithCancel(context.Background())
	h.ctx = ctx

	h.httpPrepare()

	signalHandler := make(chan os.Signal, 3)
	signal.Notify(signalHandler, os.Interrupt)

	go func() {
		<-signalHandler
		stop()
		_ = h.srv.Close()
	}()

	log.Println("server starting")

	err := h.srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		log.Println("server stopped")
		return
	}

	if err != nil {
		log.Fatal(err)
	}
}
