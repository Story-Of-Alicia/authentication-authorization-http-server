package config

import (
	"context"

	"github.com/sethvargo/go-envconfig"
)


type DiscordConfig struct {
	ClientID     string `env:"CLIENT_ID"`
	ClientSecret string `env:"CLIENT_SECRET"`
	RedirectUri  string `env:"OAUTH_REDIRECT"`
	DiscordApiUri string `env:", default=https://discord.com/api/v10"`
}

type Config struct {
	DiscordConfig *DiscordConfig
	Address       string `env:"SERVER_ADDRES, default=:8081"`
	DbDSN         string `env:"DATABASE_DSN"`
}

var configInstance *Config

func newConfig() *Config {
	config := &Config{}

	ctx := context.Background()

	if err := envconfig.Process(ctx, config); err != nil {
		panic(err)
	}

	return config
}

func GetConfigInstance() *Config {
	if (configInstance == nil) {
		configInstance = newConfig()
	}

	return configInstance
}
