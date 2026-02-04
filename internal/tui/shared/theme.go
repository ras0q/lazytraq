package shared

import "github.com/charmbracelet/lipgloss"

// Colors defines the color palette
type Colors struct {
	Primary lipgloss.Color
	Accent  lipgloss.Color
	Muted   lipgloss.Color
	Border  lipgloss.Color
}

// BorderStyles defines border styling for components
type BorderStyles struct {
	Normal  lipgloss.Style
	Focused lipgloss.Style
}

// HeaderStyles defines styling for header component
type HeaderStyles struct {
	Title    lipgloss.Style
	Host     lipgloss.Style
	Username lipgloss.Style
}

// TimelineStyles defines styling for timeline component
type TimelineStyles struct {
	Time       lipgloss.Style
	MessageBox lipgloss.Style
	Username   lipgloss.Style
	Separator  lipgloss.Style
}

// PreviewStyles defines styling for preview component
type PreviewStyles struct {
	Stamps lipgloss.Style
}

// Theme aggregates all style definitions
type Theme struct {
	Colors   Colors
	Border   BorderStyles
	Header   HeaderStyles
	Timeline TimelineStyles
	Preview  PreviewStyles
}

// DefaultTheme returns the default color scheme
func DefaultTheme() Theme {
	colors := Colors{
		Primary: lipgloss.Color("205"),
		Accent:  lipgloss.Color("240"),
		Muted:   lipgloss.Color("240"),
		Border:  lipgloss.Color("205"),
	}

	return Theme{
		Colors: colors,
		Border: BorderStyles{
			Normal:  lipgloss.NewStyle().Border(lipgloss.RoundedBorder()),
			Focused: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colors.Border),
		},
		Header: HeaderStyles{
			Title:    lipgloss.NewStyle().Bold(true).Italic(true),
			Host:     lipgloss.NewStyle().Bold(true),
			Username: lipgloss.NewStyle().Bold(true),
		},
		Timeline: TimelineStyles{
			Time: lipgloss.NewStyle().Foreground(colors.Accent).PaddingRight(1),
			MessageBox: lipgloss.NewStyle().
				BorderStyle(lipgloss.Border{Left: "│"}).
				BorderLeft(true).
				BorderForeground(colors.Muted).
				PaddingLeft(1),
			Username:  lipgloss.NewStyle().Foreground(colors.Primary).Bold(true),
			Separator: lipgloss.NewStyle().Foreground(colors.Muted),
		},
		Preview: PreviewStyles{
			Stamps: lipgloss.NewStyle().Height(8),
		},
	}
}

// WithBorder applies border style based on focus state
func (t Theme) WithBorder(content string, focused bool) string {
	if focused {
		return t.Border.Focused.Render(content)
	}
	return t.Border.Normal.Render(content)
}
