package auth

import (
	"mcp-etoolstec-editfiles/config"
	"net/http"
	"strings"
)

func CheckAuth(r *http.Request) bool {
	if config.BearerToken == "" {
		return true
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		if strings.TrimPrefix(h, "Bearer ") == config.BearerToken {
			return true
		}
	}
	t := r.URL.Query().Get("token")
	if t == "" {
		t = r.URL.Query().Get("access_token")
	}
	return t == config.BearerToken
}
