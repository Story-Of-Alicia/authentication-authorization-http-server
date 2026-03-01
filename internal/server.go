package internal

import (
	"authentication-server/internal/facade"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"
)

type AuthenticationServer struct {
	server *http.Server

	DiscordClient   *DiscordClient
	SessionProvider facade.SessionProvider
	Ctx             context.Context

	RedirectURI string
	BindAddress string
}

func (a *AuthenticationServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

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

	userID, err := a.DiscordClient.FetchUserID(code)
	if err != nil {
		log.Println("failed to fetch discord userID: ", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	whitelist := []string{"355314189117554689", "386866465648803841", "98486055170215936", "426468841284173830"}
	var found bool = false
	for _, user := range whitelist {
		if user == userID {
			found = true
		}
	}

	if !found {
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}

	session, err := a.SessionProvider.CreateSession(userID)
	if err != nil {
		log.Println("failed to create session:", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(writer, request, fmt.Sprintf("%s/?token=%s&user=%s", a.RedirectURI, session.Token, session.User), http.StatusFound)
}

func (a *AuthenticationServer) Serve() {

	context.AfterFunc(a.Ctx, func() {
		/* After the main context is cancelled, spawn a new context with 20 second timeout, which will gracefully end all connections
		* They will return 500 though, maybe TODO
		 */
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		a.server.Shutdown(ctx)
	})

	if a.SessionProvider == nil {
		panic("no session provider")
	}

	if a.DiscordClient == nil {
		panic("no discord client")
	}

	a.server = &http.Server{}
	a.server.Addr = a.BindAddress
	a.server.Handler = a

	err := a.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return
	}
	log.Println(err)
}
