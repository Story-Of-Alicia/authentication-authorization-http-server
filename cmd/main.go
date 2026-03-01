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

	return vars
}

func main() {
	/* set verbose logging */
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	ctx, cancel := context.WithCancel(context.Background())

	signalHandler := make(chan os.Signal, 3)
	signal.Notify(signalHandler, os.Interrupt)

	go func() {
		<-signalHandler
		cancel()
	}()

	vars := parseEnv()

	dc := internal.DiscordClient{
		ClientID:     vars["DISCORD_CLIENT_ID"],
		ClientSecret: vars["DISCORD_CLIENT_SECRET"],
		Oauth2URI:    vars["DISCORD_OAUTH2_URI"],
		Ctx:          ctx,
	}

	println(vars["POSTGRES_DSN"])
	provider := providers.PostgresSessionProvider{
		DSN: vars["POSTGRES_DSN"],
		Ctx: ctx,
	}

	api := internal.AuthenticationServer{
		SessionProvider: &provider,
		DiscordClient:   &dc,
		RedirectURI:     vars["REDIRECT_URL"],
		BindAddress:     "0.0.0.0:8080",
		Ctx:             ctx,
	}

	err := provider.Init()
	if err != nil {
		log.Fatal(err)
	}

	api.Serve()

	defer cancel()
}
