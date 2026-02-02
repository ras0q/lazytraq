package header

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/shared"
)

type (
	meFetchedMsg *traqapi.MyUserDetail
)

type State struct {
	me *traqapi.MyUserDetail
}

type Model struct {
	w, h        int
	apiHost     string
	traqContext *traqapiext.Context

	state State
}

func New(w, h int, apiHost string, traqContext *traqapiext.Context) *Model {
	return &Model{
		w:           w,
		h:           h,
		apiHost:     apiHost,
		traqContext: traqContext,
	}
}

func (m *Model) Init() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		me, err := m.traqContext.Me.Get(ctx, struct{}{})
		if err != nil {
			return shared.ErrorMsg(fmt.Errorf("fetch me: %w", err))
		}

		return meFetchedMsg(me)
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case meFetchedMsg:
		m.state.me = msg
	}

	return m, nil
}

var titleStyle = lipgloss.NewStyle().
	Bold(true).
	Italic(true)

func (m *Model) View() string {
	leftPart := lipgloss.JoinHorizontal(
		lipgloss.Center,
		titleStyle.Render("lazytraq"),
		" in ",
		lipgloss.NewStyle().Bold(true).Render(m.apiHost),
	)

	username := "@uknown"
	if m.state.me != nil {
		username = fmt.Sprintf("@%s", m.state.me.Name)
	}
	rightPart := lipgloss.NewStyle().
		Bold(true).
		Width(m.w - lipgloss.Width(leftPart) - 1).
		Align(lipgloss.Right).
		Render(username)

	return lipgloss.NewStyle().Height(m.h).Width(m.w).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			leftPart,
			rightPart,
		),
	)
}
