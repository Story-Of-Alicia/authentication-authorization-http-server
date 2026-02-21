package internal

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"soaauth/internal/facade"
)

type AuthenticationServer struct {
	server *http.Server

	DiscordClient   *DiscordClient
	SessionProvider *facade.SessionProvider

	RedirectURI string
	BindAddress string
}

func (a *AuthenticationServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != "GET" {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !request.URL.Query().Has("code") {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	code := request.URL.Query().Get("code")

	regex, err := regexp.Compile("^[A-z0-9]*$")
	if err != nil {
		log.Fatal(err)
	}

	if !regex.MatchString(code) {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	username, err := a.DiscordClient.FetchUsername(code)
	if err != nil {
		log.Println("failed to fetch discord username: ", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	session, err := (*a.SessionProvider).CreateSession(username)
	if err != nil {
		log.Println("failed to create session:", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(writer, request, fmt.Sprintf("%s/?token=%s&username=%s", a.RedirectURI, session.Token, username), http.StatusFound)
}

func (a *AuthenticationServer) Serve() {
	if a.SessionProvider == nil {
		panic("no session provider")
	}

	if a.DiscordClient == nil {
		panic("no discord client")
	}

	a.server = &http.Server{}
	a.server.Addr = a.BindAddress

	err := a.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return
	}
	log.Println(err)
}
