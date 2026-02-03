package main


import (
	"soaauth/internal/api"
)



func main() {
	server, err := api.NewAPIServer(":8081");

	if err != nil {
		panic(err)
	}


	server.Serv();
}
