package main

import (
	"authentication-server/internal"
	"authentication-server/internal/providers"
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
)

func parseEnv() map[string]string {
	env := os.Environ()
	vars := make(map[string]string)

	for _, e := range env {
		key, val, found := strings.Cut(e, "=")
		if !found {
			continue
		}
		vars[key] = val
	}

	_, ok := vars["DISCORD_CLIENT_ID"]
	if !ok {
		log.Fatal("DISCORD_CLIENT_ID is not set")
	}

	_, ok = vars["DISCORD_CLIENT_SECRET"]
	if !ok {
		log.Fatal("DISCORD_CLIENT_SECRET is not set")
	}

	_, ok = vars["DISCORD_OAUTH2_URI"]
	if !ok {
		log.Fatal("DISCORD_OAUTH2_URI is not set")
	}

	_, ok = vars["POSTGRES_DSN"]
	if !ok {
		log.Fatal("POSTGRES_DSN is not set")
	}

	_, ok = vars["REDIRECT_URL"]
	if !ok {
		log.Fatal("REDIRECT_URL is not set")
	}

	_, ok = vars["REDIRECT_URL"]
	if !ok {
		log.Fatal("REDIRECT_URL is not set")
	}

	_, ok = vars["BIND_ADDRESS"]
	if !ok {
		log.Fatal("BIND_ADDRESS is not set")
	}

	return vars
}

func main() {
	/* set verbose logging */
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Ldate | log.Ltime)

	ctx, cancel := context.WithCancel(context.Background())

	signalHandler := make(chan os.Signal, 3)
	signal.Notify(signalHandler, os.Interrupt)

	go func() {
		<-signalHandler
		log.Println("Authentication server is shutting down")
		cancel()
	}()

	vars := parseEnv()

	dc := internal.DiscordClient{
		ClientID:     vars["DISCORD_CLIENT_ID"],
		ClientSecret: vars["DISCORD_CLIENT_SECRET"],
		Oauth2URI:    vars["DISCORD_OAUTH2_URI"],
		Ctx:          ctx,
	}

	provider := providers.PostgresSessionProvider{
		DSN: vars["POSTGRES_DSN"],
		Ctx: ctx,
	}

	api := internal.AuthenticationServer{
		SessionProvider: &provider,
		DiscordClient:   &dc,
		RedirectURI:     vars["REDIRECT_URL"],
		BindAddress:     vars["BIND_ADDRESS"],
		Ctx:             ctx,
	}

	err := provider.Init()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Authentication server is listening on %s\n", api.BindAddress)
	api.Serve()

	defer cancel()
}
