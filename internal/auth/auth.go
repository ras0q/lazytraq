package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	traqoauth2 "github.com/traPtitech/go-traq-oauth2"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

var (
	configDir, _ = os.UserConfigDir()
	ltConfigDir  = filepath.Join(configDir, "lazytraq")
	ltHostFile   = "hosts.json"

	keyringUser = "lazytraq-user"

	errTokenNotFound = errors.New("token not found")
)

type TokenStore int

const (
	TokenStoreUnknown TokenStore = iota
	TokenStoreKeyring
	TokenStoreFile
	TokenStoreWeb
)

func GetToken(ctx context.Context, apiHost string, authURLCh chan<- string) (*oauth2.Token, TokenStore, error) {
	token, err := getTokenFromKeyring(keyringService(apiHost), keyringUser)
	if err == nil {
		return token, TokenStoreKeyring, nil
	}

	token, err = getTokenFromFile(apiHost)
	if err == nil {
		return token, TokenStoreFile, nil
	}

	if errors.Is(err, errTokenNotFound) {
		token, err := getTokenFromWeb(ctx, apiHost, authURLCh)
		if err == nil {
			return token, TokenStoreWeb, nil
		}

		return nil, TokenStoreUnknown, fmt.Errorf("get token from web: %w", err)
	}

	return nil, TokenStoreUnknown, fmt.Errorf("get token from file: %w", err)
}

func getTokenFromKeyring(service, username string) (*oauth2.Token, error) {
	token, err := keyring.Get(service, username)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, errTokenNotFound
		}

		return nil, fmt.Errorf("get token from keyring: %w", err)
	}

	return &oauth2.Token{
		AccessToken: token,
	}, nil
}

func getTokenFromFile(host string) (*oauth2.Token, error) {
	f, err := os.OpenInRoot(ltConfigDir, ltHostFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errTokenNotFound
		}

		return nil, fmt.Errorf("open file (%s/%s): %w", ltConfigDir, ltHostFile, err)
	}

	var tokens map[string]string
	if err := json.NewDecoder(f).Decode(&tokens); err != nil {
		return nil, fmt.Errorf("decode file (%s/%s) to json: %w", ltConfigDir, ltHostFile, err)
	}

	token, ok := tokens[host]
	if !ok {
		return nil, errTokenNotFound
	}

	return &oauth2.Token{
		AccessToken: token,
	}, nil
}

func getTokenFromWeb(ctx context.Context, apiHost string, authURLCh chan<- string) (*oauth2.Token, error) {
	state := "state" // TODO: generate random state

	endpoint, err := traqoauth2.New(fmt.Sprintf("https://%s/api/v3", apiHost))
	if err != nil {
		return nil, fmt.Errorf("create oauth2 endpoint: %w", err)
	}

	oauth2Config := oauth2.Config{
		ClientID:    "E4d5xiUOC0I803NjujtuDOQKBHN4b2GWj4oo",
		Endpoint:    endpoint,
		RedirectURL: "http://localhost:8080",
		Scopes: []string{
			traqoauth2.ScopeRead,
			traqoauth2.ScopeWrite,
		},
	}
	authURL := oauth2Config.AuthCodeURL(state)

	// _, _ = w.Write([]byte(authURL))
	authURLCh <- authURL

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

func SetToken(apiHost string, token *oauth2.Token) (TokenStore, error) {
	if token == nil {
		return TokenStoreUnknown, fmt.Errorf("token is nil")
	}

	keyringErr := setTokenToKeyring(keyringService(apiHost), keyringUser, token.AccessToken)
	if keyringErr == nil {
		return TokenStoreKeyring, nil
	}

	fileErr := setTokenToFile(apiHost, token.AccessToken)
	if fileErr == nil {
		return TokenStoreFile, nil
	}

	return TokenStoreUnknown, fmt.Errorf("set token to keyring: %v; set token to file: %w", keyringErr, fileErr)
}

func setTokenToKeyring(service, username, token string) error {
	if err := keyring.Set(service, username, token); err != nil {
		return fmt.Errorf("set token to keyring: %w", err)
	}

	return nil
}

func setTokenToFile(host, token string) error {
	if err := os.MkdirAll(ltConfigDir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	root, err := os.OpenRoot(ltConfigDir)
	if err != nil {
		return fmt.Errorf("open config dir: %w", err)
	}

	tokens := map[string]string{}
	if data, err := root.ReadFile(ltHostFile); err == nil {
		if err := json.Unmarshal(data, &tokens); err != nil {
			return fmt.Errorf("decode file (%s) to json: %w", ltHostFile, err)
		}
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read file (%s): %w", ltHostFile, err)
		}
	}

	tokens[host] = token

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("encode tokens to json: %w", err)
	}

	if err := root.WriteFile(ltHostFile, data, 0600); err != nil {
		return fmt.Errorf("write file (%s): %w", ltHostFile, err)
	}

	return nil
}

func keyringService(apiHost string) string {
	return fmt.Sprintf("lazytraq-%s", apiHost)
}
