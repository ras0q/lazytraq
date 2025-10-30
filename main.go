package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ras0q/lazytraq/internal/model/root"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
)

func main() {
	if err := runProgram(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func runProgram() error {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return fmt.Errorf("get terminal size: %w", err)
	}

	// NOTE: decrease padding
	h = h - 2

	model, err := root.New(w, h)
	if err != nil {
		return fmt.Errorf("create root model: %w", err)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	errCh := make(chan error)

	eg := errgroup.Group{}
	eg.Go(func() error {
		return <-errCh
	})

	eg.Go(func() error {
		if _, err := p.Run(); err != nil {
			return err
		}

		errCh <- nil

		return nil
	})

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("run program: %w", err)
	}

	return nil
}
