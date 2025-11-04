package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/root"
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

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run tea program: %w", err)
	}

	close(model.ErrCh)

	if err := <-model.ErrCh; err != nil {
		return fmt.Errorf("application error: %w", err)
	}

	return nil
}
