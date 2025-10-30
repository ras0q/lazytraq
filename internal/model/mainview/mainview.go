package mainview

import tea "github.com/charmbracelet/bubbletea"

type MainViewModel struct {
	w, h int
}

var _ tea.Model = (*MainViewModel)(nil)

func New(w, h int) *MainViewModel {
	return &MainViewModel{
		w: w,
		h: h,
	}
}

func (m *MainViewModel) Init() tea.Cmd {
	return nil
}

func (m *MainViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *MainViewModel) View() string {
	return "Main View"
}
