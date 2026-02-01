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
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const mysqlDsn = "test:1234qweRty@tcp(127.0.0.1:3306)/test"
const addr = "127.0.0.1:8080"

const DiscordApi = "https://discord.com/api/v10"
const ClientId = "1272602862043795586"
const RedirectUri = "http://localhost/api/callback"

type server struct {
	db     *sql.DB
	ctx    context.Context
	srv    *http.Server
	secret string
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

func (h *server) dbPrepare() error {
	ctx, cancel := context.WithTimeout(h.ctx, 5*time.Second)
	defer cancel()

	_, err := h.db.ExecContext(ctx,
		"CREATE TABLE IF NOT EXISTS `users` ("+
			"email VARCHAR(255) NOT NULL,"+
			"password varchar(32) NOT NULL,"+
			"username varchar(32) PRIMARY KEY);",
	)
	if err != nil {
		return err
	}

	_, err = h.db.ExecContext(ctx,
		"CREATE TABLE IF NOT EXISTS `sessions` ("+
			"token VARCHAR(64) NOT NULL,"+
			"username VARCHAR(32) PRIMARY KEY,"+
			"expires_at TIMESTAMP);",
	)
	if err != nil {
		return err
	}

	return nil
}

func (h *server) dbConnect() error {
	db, err := sql.Open("mysql", mysqlDsn)
	if err != nil {
		return err
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	h.db = db

	return nil
}

func (h *server) dbCreateSession(username string) (string, error) {
	ctx, cancel := context.WithTimeout(h.ctx, 5*time.Second)
	defer cancel()

	token := generateToken(32)
	expiry := time.Now().Add(time.Hour)

	var exists bool
	err := h.db.QueryRowContext(ctx,
		"SELECT EXISTS("+
			"SELECT 1 FROM `sessions` WHERE `username` = ?)",
		username,
	).Scan(&exists)

	if err != nil {
		return "", err
	}

	if exists {
		_, err := h.db.ExecContext(ctx,
			"UPDATE `sessions` SET `token` = ?, `expires_at` = ? WHERE username = ?", token, expiry, username)
		if err != nil {
			return "", err
		}
	} else {
		_, err := h.db.ExecContext(ctx,
			"INSERT `sessions` (`username`, `token`, `expires_at`) VALUES (?, ?, ?)", username, token, expiry)
		if err != nil {
			return "", err
		}
	}

	return token, nil
}

func discordGetUsername(secret string, code string) (string, error) {
	client := http.Client{}

	var payload map[string]interface{}
	var err error
	var buf []byte
	var response *http.Response

	form := url.Values{
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"client_id":     {ClientId},
		"client_secret": {secret},
		"redirect_uri":  {RedirectUri},
	}

	response, err = client.PostForm(fmt.Sprintf("%s/oauth2/token", DiscordApi), form)
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
		return "", err
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
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/users/@me", DiscordApi), nil)
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

func (h *server) httpHandleCallback(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !req.URL.Query().Has("code") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	code := req.URL.Query().Get("code")

	regex, err := regexp.Compile("^[A-z0-9]$")
	if err != nil {
		log.Fatal(err)
	}

	if !regex.MatchString(code) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	username, err := discordGetUsername(h.secret, code)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	token, err := h.dbCreateSession(username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, fmt.Sprintf("https://storyofalicia.com/?token=%s", token), http.StatusFound)
}

func (h *server) httpPrepare() {
	h.srv = &http.Server{}
	h.srv.Addr = addr

	http.HandleFunc("/callback", h.httpHandleCallback)
}

func main() {
	/* set verbose logging */
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var h = server{}
	ctx, stop := context.WithCancel(context.Background())
	h.secret = os.Args[1]
	h.ctx = ctx

	err := h.dbConnect()
	if err != nil {
		log.Fatal(err)
	}

	err = h.dbPrepare()
	if err != nil {
		log.Fatal(err)
	}

	h.httpPrepare()

	signalHandler := make(chan os.Signal, 3)
	signal.Notify(signalHandler, os.Interrupt)

	go func() {
		<-signalHandler
		stop()
		_ = h.srv.Close()
		_ = h.db.Close()
	}()

	log.Println("server starting")

	err = h.srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		log.Println("server stopped")
		return
	}

	if err != nil {
		log.Fatal(err)
	}
}
