package api


type Reponse struct {
	Status int `json:"status"`
	Message string `json:"message"`
	Data any `json:"data"`
}


type TokenResponse struct {
	Token string `json:"token"`
}
