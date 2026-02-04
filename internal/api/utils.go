package api

import (
	"encoding/json"
	"soaauth/internal/types"
	"net/http"
)

type apiFunc func(http.ResponseWriter, *http.Request) *types.APIError

func JSONResponse(w http.ResponseWriter, response Reponse) error {
	w.WriteHeader(response.Status)
	w.Header().Add("Content-Type", "application/json")

	return json.NewEncoder(w).Encode(response)
}

func MakeHTTPFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			JSONResponse(w, Reponse{
				Status:  err.Code,
				Message: err.Message,
				Data:    nil,
			})
		}
	}
}
