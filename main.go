package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ras0q/goalie"
	"github.com/ras0q/lazytraq/internal/auth"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui"
	"golang.org/x/term"
)

func main() {
	ctx := context.Background()
	if err := runProgram(ctx); err != nil {
		panic(err)
	}
}

func runProgram(ctx context.Context) (err error) {
	g := goalie.New()
	defer g.Collect(&err)

	var isDebugMode bool
	if v, ok := os.LookupEnv("LAZYTRAQ_DEBUG"); ok && v != "" {
		isDebugMode = true
	}

	if isDebugMode {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	logDir := cmp.Or(
		os.Getenv("XDG_STATE_HOME"),
		path.Join(os.Getenv("HOME"), ".local", "state"),
	)
	logFilePath := path.Join(logDir, "lazytraq", "lazytraq.log")
	if err := os.MkdirAll(path.Dir(logFilePath), 0o755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer g.Guard(logFile.Close)

	slog.SetDefault(
		slog.New(slog.NewJSONHandler(logFile, nil)),
	)

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

	opts := []tea.ProgramOption{tea.WithAltScreen()}
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
