package main

import (
	"soaauth/internal/api"
	"soaauth/internal/config"
)



func main() {
	config := config.GetConfigInstance()

	server, err := api.NewAPIServer(config.Address);

	if err != nil {
		panic(err)
	}

	server.Serv();
}
