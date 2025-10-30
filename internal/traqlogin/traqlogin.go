package traqlogin

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"

	traqoauth2 "github.com/traPtitech/go-traq-oauth2"
	"golang.org/x/oauth2"
)

var (
	oauth2Config = oauth2.Config{
		ClientID:    "QAQIPJ7xA8ZTnK5qP0uxhleF0xasudkazUJ2",
		Endpoint:    traqoauth2.Prod,
		RedirectURL: "http://localhost:8080",
		Scopes: []string{
			traqoauth2.ScopeRead,
			traqoauth2.ScopeWrite,
		},
	}
)

func GetToken(ctx context.Context, w io.Writer) (*oauth2.Token, error) {
	state := "state" // TODO: generate random state

	authURL := oauth2Config.AuthCodeURL(state)
	_, _ = w.Write([]byte(authURL))

	codeCh, err := startCallbackServer(":8080")
	if err != nil {
		return nil, fmt.Errorf("start callback server: %w", err)
	}

	code := <-codeCh

	token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code for token: %w", err)
	}

	return token, nil
}

func startCallbackServer(addr string) (chan string, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	codeCh := make(chan string)

	//nolint:errcheck
	go http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		codeCh <- r.FormValue("code")
		listener.Close()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Login successful!"))
	}))

	return codeCh, nil
}
