package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const mysqlDsn = "test:1234qweRty@tcp(127.0.0.1:3306)/test"
const addr = "127.0.0.1:8080"

const maxUsernameLength = 32
const maxPasswordLength = 32

type server struct {
	db  *sql.DB
	ctx context.Context
	srv *http.Server
}

func usernameSanity(val string) bool {
	if len(val) > maxUsernameLength {
		return false
	}

	p, err := regexp.Compile("^\\w+$")
	if err != nil {
		panic(err)
	}
	return p.MatchString(val)
}

func passwordSanity(val string) bool {
	if len(val) > maxPasswordLength {
		return false
	}

	p, err := regexp.Compile("^\\S+$")
	if err != nil {
		panic(err)
	}
	return p.MatchString(val)
}

func generateToken(size int) string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLKMNOPQRSTVWXYZ0123456789"
	b := make([]byte, size)
	rand.Read(b)

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

	_, err = h.db.ExecContext(ctx,
		"CREATE TABLE IF NOT EXISTS `verifications` ("+
			"token VARCHAR(64) NOT NULL,"+
			"email VARCHAR(255) NOT NULL,"+
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

func (h *server) dbAuthenticateUser(username string, password string) (bool, error) {
	ctx, cancel := context.WithTimeout(h.ctx, 5*time.Second)
	defer cancel()

	var exists bool
	err := h.db.QueryRowContext(ctx,
		"SELECT EXISTS("+
			"SELECT 1 FROM `users` WHERE `username` = ? AND `password` = ?)",
		username, password,
	).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}

func (h *server) dbUserExists(username string) (bool, error) {
	ctx, cancel := context.WithTimeout(h.ctx, 5*time.Second)
	defer cancel()

	var exists bool
	err := h.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM `users` WHERE `username` = ?)", username).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (h *server) dbCreateUser(username string, password string) error {
	ctx, cancel := context.WithTimeout(h.ctx, 5*time.Second)
	defer cancel()

	_, err := h.db.ExecContext(ctx,
		"INSERT INTO `users` (`username`, `password`) VALUES (?, ?)", username, password)
	if err != nil {
		return err
	}

	return nil
}

func (h *server) httpHandleRegister(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := req.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !req.Form.Has("password") && !req.Form.Has("username") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	username := req.Form.Get("username")
	if !usernameSanity(username) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	password := req.Form.Get("password")
	if !passwordSanity(password) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	v, err := h.dbUserExists(username)
	if v {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("username taken"))
		return
	}

	hash := sha256.New()
	hash.Write([]byte(password))
	bs := hash.Sum(nil)

	hashString := fmt.Sprintf("%x", bs)
	err = h.dbCreateUser(username, hashString)
	if err != nil {
		log.Printf("Error while processing register request for username: '%s'\n", username)
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}

func (h *server) httpHandleLogin(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := req.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !req.Form.Has("password") && !req.Form.Has("username") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	username := req.Form.Get("username")
	if !usernameSanity(username) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	password := req.Form.Get("password")
	if !passwordSanity(password) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hash := sha256.New()
	hash.Write([]byte(password))
	bs := hash.Sum(nil)

	hashString := fmt.Sprintf("%x", bs)

	v, err := h.dbAuthenticateUser(username, hashString)
	if err != nil {
		log.Printf("Error while processing login request for username: '%s'\n", username)
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !v {
		w.Write([]byte("invalid credentials"))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token, err := h.dbCreateSession(username)
	if err != nil {
		log.Printf("Error while processing login request for username: '%s'\n", username)
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(token))

	return
}

func (h *server) httpPrepare() {
	h.srv = &http.Server{}
	h.srv.Addr = addr

	http.HandleFunc("/login", h.httpHandleLogin)
	http.HandleFunc("/register", h.httpHandleRegister)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	ctx, stop := context.WithCancel(context.Background())

	var h = server{}
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
		h.srv.Close()
		h.db.Close()
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
