package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ras0q/lazytraq/internal/auth"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui"
	"golang.org/x/term"
)

func main() {
	ctx := context.Background()
	if err := runProgram(ctx); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func runProgram(ctx context.Context) error {
	apiHost := "q-dev.trapti.tech"
	securitySource, err := loginToTraq(ctx, apiHost)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return fmt.Errorf("get terminal size: %w", err)
	}

	// NOTE: decrease padding
	h = h - 2

	model, err := tui.NewAppModel(w, h, apiHost, securitySource)
	if err != nil {
		return fmt.Errorf("create root model: %w", err)
	}


	opts := []tea.ProgramOption{}
	if os.Getenv("DEBUG") == "" {
		opts = append(opts, tea.WithAltScreen())
	}

	p := tea.NewProgram(model, opts...)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run tea program: %w", err)
	}

	if len(model.Errors) > 0 {
		return fmt.Errorf("application errors: %w", errors.Join(model.Errors...))
	}

	return nil
}

func loginToTraq(ctx context.Context, apiHost string) (*traqapiext.SecuritySource, error) {
	authURLCh := make(chan string, 1)
	defer close(authURLCh)

	go func() {
		for authURL := range authURLCh {
			fmt.Printf(
				"Please open the following URL in your browser to authenticate:\n\n%s\n\n",
				authURL,
			)
		}
	}()

	token, tokenStore, err := auth.GetToken(ctx, apiHost, authURLCh)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	if tokenStore == auth.TokenStoreFile {
		slog.WarnContext(ctx, "got token from file, consider using keyring")
	}

	if tokenStore == auth.TokenStoreWeb {
		tokenStore, err = auth.SetToken(apiHost, token)
		if err != nil {
			return nil, fmt.Errorf("set token: %w", err)
		}

		if tokenStore == auth.TokenStoreFile {
			slog.WarnContext(ctx, "saved token to file, consider using keyring")
		}
	}

	return traqapiext.NewSecuritySource(token.AccessToken), nil
}
