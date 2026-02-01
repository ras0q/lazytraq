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
	var isDebugMode bool
	if v, ok := os.LookupEnv("LAZYTRAQ_DEBUG"); ok && v != "" {
		isDebugMode = true
	}

	if isDebugMode {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	apiHost := "q-dev.trapti.tech"

	slog.DebugContext(ctx, "starting lazytraq", "apiHost", apiHost)

	securitySource, err := loginToTraq(ctx, apiHost)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	slog.DebugContext(ctx, "logged in successfully", "apiHost", apiHost)

	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return fmt.Errorf("get terminal size: %w", err)
	}

	// NOTE: decrease padding
	h = h - 2

	slog.DebugContext(ctx, "got terminal size", "width", w, "height", h)

	model, err := tui.NewAppModel(w, h, apiHost, securitySource)
	if err != nil {
		return fmt.Errorf("create root model: %w", err)
	}

	slog.DebugContext(ctx, "created root model")

	opts := []tea.ProgramOption{}
	if !isDebugMode {
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
