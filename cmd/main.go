package main

import (
	"log"
	"soaauth/internal"
	"soaauth/internal/providers"
)

func main() {
	/* set verbose logging */
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	dc := internal.DiscordClient{
		ClientID:     "clientID",
		ClientSecret: "clientSecret",
		Oauth2URI:    "oauthUri",
	}

	provider := providers.FlatFileSessionProvider{
		WorkDir: "/opt/",
	}

	api := internal.AuthenticationServer{
		SessionProvider: &provider,
		DiscordClient:   &dc,
		RedirectURI:     "https://storyofalicia.com/play",
		BindAddress:     "0.0.0.0:8080",
	}

	api.Serve()
}
