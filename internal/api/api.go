package api

import (
	"log"
	"net/http"
	"soaauth/internal/database"
	"soaauth/internal/types"
)

type APIServer struct {
	db   *database.DB
	serv *http.Server

	addr string
}

func NewAPIServer(addr string) (APIServer, error) {
	db, err := database.CreateDb()
	if err != nil {
		return APIServer{}, err
	}
	return APIServer{
		db:   &db,
		addr: addr,
		serv: &http.Server{
			Addr: addr,
		},
	}, nil
}

func (s *APIServer) Serv() {
	log.Println("Start server at addr: " + s.addr)

	http.HandleFunc("/callback", MakeHTTPFunc(s.handleDiscordCallback))

	log.Fatal(s.serv.ListenAndServe())
}

func (s *APIServer) handleDiscordCallback(w http.ResponseWriter, r *http.Request) *types.APIError {
	if r.Method != http.MethodGet {
		return &types.APIError{
			Code:    http.StatusMethodNotAllowed,
			Message: "Method not allowed",
		}
	}

	return nil
}

// package api
//
// import (
// 	"context"
// 	"crypto/rand"
// 	"database/sql"
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"io"
// 	"log"
// 	"net/http"
// 	"net/url"
// 	"os"
// 	"os/signal"
// 	"regexp"
// 	"strings"
// 	"time"
//
// 	_ "github.com/go-sql-driver/mysql"
// )
//
// const DiscordApiURI = "https://discord.com/api/v10"
//
// type server struct {
// 	db  *sql.DB
// 	ctx context.Context
// 	srv *http.Server
//
// 	secret string
// 	id     string
// 	dbDsn  string
// 	addr   string
// 	redir  string
// }
//
// func generateToken(size int) string {
// 	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLKMNOPQRSTVWXYZ0123456789"
// 	b := make([]byte, size)
// 	r, err := rand.Read(b)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	if r != size {
// 		log.Fatal("failed to generate random token")
// 	}
//
// 	for i := range b {
// 		b[i] = chars[b[i]%byte(len(chars))]
// 	}
//
// 	return string(b)
// }
//
// func (s *server) dbPrepare() error {
// 	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
// 	defer cancel()
//
// 	_, err := s.db.ExecContext(ctx,
// 		"CREATE TABLE IF NOT EXISTS `users` ("+
// 			"email VARCHAR(255) NOT NULL,"+
// 			"username varchar(32) PRIMARY KEY);",
// 	)
// 	if err != nil {
// 		return err
// 	}
//
// 	_, err = s.db.ExecContext(ctx,
// 		"CREATE TABLE IF NOT EXISTS `sessions` ("+
// 			"token VARCHAR(64) NOT NULL,"+
// 			"username VARCHAR(32) PRIMARY KEY,"+
// 			"expires_at TIMESTAMP);",
// 	)
// 	if err != nil {
// 		return err
// 	}
//
// 	return nil
// }
//
// func (s *server) dbConnect() error {
// 	db, err := sql.Open("mysql", s.dbDsn)
// 	if err != nil {
// 		return err
// 	}
//
// 	db.SetConnMaxLifetime(time.Minute * 3)
// 	db.SetMaxOpenConns(10)
// 	db.SetMaxIdleConns(10)
//
// 	s.db = db
//
// 	return nil
// }
//
// func (s *server) dbCreateSession(username string) (string, error) {
// 	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
// 	defer cancel()
//
// 	token := generateToken(32)
// 	expiry := time.Now().Add(time.Hour)
//
// 	var exists bool
// 	err := s.db.QueryRowContext(ctx,
// 		"SELECT EXISTS("+
// 			"SELECT 1 FROM `sessions` WHERE `username` = ?)",
// 		username,
// 	).Scan(&exists)
//
// 	if err != nil {
// 		return "", err
// 	}
//
// 	if exists {
// 		_, err := s.db.ExecContext(ctx,
// 			"UPDATE `sessions` SET `token` = ?, `expires_at` = ? WHERE username = ?", token, expiry, username)
// 		if err != nil {
// 			return "", err
// 		}
// 	} else {
// 		_, err := s.db.ExecContext(ctx,
// 			"INSERT `sessions` (`username`, `token`, `expires_at`) VALUES (?, ?, ?)", username, token, expiry)
// 		if err != nil {
// 			return "", err
// 		}
// 	}
//
// 	return token, nil
// }
//
// func (s *server) discordGetUsername(code string) (string, error) {
// 	client := http.Client{}
//
// 	var payload map[string]interface{}
// 	var err error
// 	var buf []byte
// 	var response *http.Response
//
// 	form := url.Values{
// 		"code":          {code},
// 		"grant_type":    {"authorization_code"},
// 		"client_id":     {s.id},
// 		"client_secret": {s.secret},
// 		"redirect_uri":  {s.redir},
// 	}
//
// 	response, err = client.PostForm(fmt.Sprintf("%s/oauth2/token", DiscordApiURI), form)
// 	if err != nil {
// 		log.Println(err)
// 		return "", err
// 	}
//
// 	buf, err = io.ReadAll(response.Body)
// 	_ = response.Body.Close()
// 	if err != nil {
// 		log.Println(err)
// 		return "", err
// 	}
//
// 	if response.StatusCode != http.StatusOK {
// 		log.Println("bad response status code from oauth2")
// 		log.Println(string(buf))
// 		return "", err
// 	}
//
// 	err = json.Unmarshal(buf, &payload)
// 	if err != nil {
// 		log.Println(err)
// 		return "", err
// 	}
//
// 	if payload["access_token"] == nil {
// 		log.Println("missing 'access_token' field from discord api response")
// 		log.Println(string(buf))
// 		return "", err
// 	}
//
// 	token := payload["access_token"].(string)
// 	request, err := http.NewRequest("GET", fmt.Sprintf("%s/users/@me", DiscordApiURI), nil)
// 	if err != nil {
// 		log.Println(err)
// 		return "", err
// 	}
//
// 	request.Header = http.Header{
// 		"Authorization": []string{fmt.Sprintf("Bearer %s", token)},
// 	}
//
// 	response, err = client.Do(request)
// 	if err != nil {
// 		log.Println(err)
// 		return "", err
// 	}
//
// 	buf, err = io.ReadAll(response.Body)
// 	_ = response.Body.Close()
// 	if err != nil {
// 		log.Println(err)
// 		return "", err
// 	}
//
// 	err = json.Unmarshal(buf, &payload)
// 	if err != nil {
// 		log.Println(err)
// 		return "", err
// 	}
//
// 	if payload["username"] == nil {
// 		log.Println("missing 'username' field from discord api response")
// 		log.Println(string(buf))
// 		return "", err
// 	}
//
// 	return payload["username"].(string), nil
// }
//
// func (s *server) httpHandleCallback(w http.ResponseWriter, req *http.Request) {
// 	if req.Method != "GET" {
// 		w.WriteHeader(http.StatusMethodNotAllowed)
// 		return
// 	}
//
// 	if !req.URL.Query().Has("code") {
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}
//
// 	code := req.URL.Query().Get("code")
//
// 	regex, err := regexp.Compile("^[A-z0-9]*$")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	if !regex.MatchString(code) {
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}
//
// 	username, err := s.discordGetUsername(code)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		return
// 	}
//
// 	token, err := s.dbCreateSession(username)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		return
// 	}
//
// 	http.Redirect(w, req, fmt.Sprintf("https://storyofalicia.com/?token=%s", token), http.StatusFound)
// }
//
// func (s *server) httpPrepare() {
// 	s.srv = &http.Server{}
// 	s.srv.Addr = s.addr
//
// 	http.HandleFunc("/callback", s.httpHandleCallback)
// }
//
// func (s *server) parseEnv() {
// 	env := os.Environ()
// 	for _, e := range env {
// 		k := strings.Split(e, "=")[0]
// 		v := strings.Split(e, "=")[1]
// 		switch {
// 		case k == "CLIENT_SECRET":
// 			s.secret = v
// 		case k == "CLIENT_ID":
// 			s.id = v
// 		case k == "DATABASE":
// 			s.dbDsn = v
// 		case k == "ADDRESS":
// 			s.addr = v
// 		case k == "REDIRECTION":
// 			s.redir = v
// 		}
// 	}
// }
//
// /* environment parameters:
//  * CLIENT_SECRET -> discord application secret
//  * CLIENT_ID -> discord application id
//  * DATABASE -> database DSN
//  * ADDRESS -> http server bind address
//  * REDIRECTION -> oauth2 redirection url
//  */
// func main() {
// 	/* set verbose logging */
// 	log.SetFlags(log.LstdFlags | log.Lshortfile)
//
// 	var h = server{}
// 	h.parseEnv()
//
// 	ctx, stop := context.WithCancel(context.Background())
// 	h.ctx = ctx
//
// 	err := h.dbConnect()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	err = h.dbPrepare()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	h.httpPrepare()
//
// 	signalHandler := make(chan os.Signal, 3)
// 	signal.Notify(signalHandler, os.Interrupt)
//
// 	go func() {
// 		<-signalHandler
// 		stop()
// 		_ = h.srv.Close()
// 		_ = h.db.Close()
// 	}()
//
// 	log.Println("server starting")
//
// 	err = h.srv.ListenAndServe()
// 	if errors.Is(err, http.ErrServerClosed) {
// 		log.Println("server stopped")
// 		return
// 	}
//
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }
