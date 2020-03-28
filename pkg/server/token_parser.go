package server

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"strings"
)

// AuthData wraps basic auth pairs
type AuthData struct {
	Username string
	Password string
}

func parseToken(r *http.Request) (*AuthData, error) {
	auth := r.Header.Get("Authorization")
	if 0 == len(auth) {
		return nil, nil
	}
	if !strings.Contains(auth, "Basic ") {
		// Unsupported auth scheme, for example, NTLM
		return nil, nil
	}
	encodedToken := strings.Replace(auth, "Basic ", "", -1)
	b, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		return nil, err
	}
	pair := strings.SplitN(bytes.NewBuffer(b).String(), ":", 2)
	if len(pair) != 2 {
		return nil, nil
	}
	username := strings.TrimSpace(pair[0])
	password := strings.TrimSpace(pair[1])
	if len(username) == 0 || len(password) == 0 {
		return nil, nil
	}
	return &AuthData{Username: username, Password: password}, nil
}
