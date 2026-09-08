package auth

import (
	"net/http"

	"mcp-etoolstec-editfiles/config"
)

func CheckAuth(r *http.Request) bool {
	if config.BearerToken == "" {
		return true
	}
	auth := r.Header.Get("Authorization")
	return auth == "Bearer "+config.BearerToken
}
