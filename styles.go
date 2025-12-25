package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"} // Green
	warning   = lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#F55385"} // Pink/Red
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"} // Purple

	// Styles
	docStyle = lipgloss.NewStyle().Padding(1, 2)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Bold(true).
			MarginBottom(1)

	clockStyle = lipgloss.NewStyle().
			Foreground(special).
			Bold(true).
			Align(lipgloss.Center)

	statLabelStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Transform(strings.ToUpper).
			Width(16)

	statValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF")).
			Bold(true)
)

func renderStat(label string, val string) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Bottom,
		statLabelStyle.Render(label),
		statValueStyle.Render(val),
	)
}

// FIX: Change 'lipgloss.Color' to 'lipgloss.TerminalColor' here
func renderBar(percent float64, width int, color lipgloss.TerminalColor) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int(float64(width) * (percent / 100))
	empty := width - filled

	// Protect against negative slicing if calculation floats slightly off
	if empty < 0 {
		empty = 0
	}
	if filled > width {
		filled = width
	}

	barChar := "█"
	emptyChar := "░"

	return lipgloss.NewStyle().Foreground(color).Render(strings.Repeat(barChar, filled)) +
		lipgloss.NewStyle().Foreground(subtle).Render(strings.Repeat(emptyChar, empty))
}
