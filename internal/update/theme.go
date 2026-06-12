package update

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/jamesMcMeex/waveshell/internal/config"
)

type Theme struct {
	BG     lipgloss.Color
	FG     lipgloss.Color
	Accent lipgloss.Color
	Muted  lipgloss.Color
}

func ResolveTheme(cfg *config.Config) Theme {
	var t config.ThemeConfig
	if cfg != nil {
		var err error
		t, err = cfg.Theme()
		if err != nil {
			t = config.ThemeConfig{
				BG:     "#1e2030",
				FG:     "#c8d3f5",
				Accent: "#82aaff",
				Muted:  "#545c7e",
			}
		}
	} else {
		t = config.ThemeConfig{
			BG:     "#1e2030",
			FG:     "#c8d3f5",
			Accent: "#82aaff",
			Muted:  "#545c7e",
		}
	}
	return Theme{
		BG:     lipgloss.Color(t.BG),
		FG:     lipgloss.Color(t.FG),
		Accent: lipgloss.Color(t.Accent),
		Muted:  lipgloss.Color(t.Muted),
	}
}

func (t Theme) PaneBorder(active bool) lipgloss.Style {
	if active {
		return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.Accent)
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.Muted)
}

func (t Theme) PaneTitle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
}

func (t Theme) CursorLine() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.FG).Background(t.Accent)
}

func (t Theme) NormalLine() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.FG)
}

func (t Theme) StatusBar() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Muted)
}

func (t Theme) KeyHintBar() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Muted)
}

func (t Theme) FormatBadge() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Accent)
}

func (t Theme) MutedText() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Muted)
}

func (t Theme) HelpContent() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.FG)
}

func (t Theme) HelpTitle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
}

func (t Theme) HelpKey() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Accent)
}

func (t Theme) NowPlaying() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.FG)
}

func (t Theme) NowPlayingMuted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Muted)
}

func (t Theme) NowPlayingAccent() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
}
