package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
)

type tickMsg time.Time

type model struct {
	width, height int
	time          time.Time
	sys           systemMsg
	gh            githubMsg
	sp            spotifyMsg
	spArt         string
}

func main() {
	_ = godotenv.Load()
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) }),
		fetchSystemCmd(),
		fetchGithubCmd(),
		fetchSpotifyCmd(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		m.time = time.Time(msg)
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
	case systemMsg:
		m.sys = msg
		return m, tea.Tick(2*time.Second, func(_ time.Time) tea.Msg { return fetchSystem() })
	case githubMsg:
		m.gh = msg
		return m, tea.Tick(60*time.Second, func(_ time.Time) tea.Msg { return fetchGithub() })
	case spotifyMsg:
		var cmd tea.Cmd
		if msg.imageUrl != m.sp.imageUrl {
			cmd = downloadImageCmd(msg.imageUrl)
		} else {
			msg.imageUrl = m.sp.imageUrl
		}
		m.sp = msg
		if cmd != nil {
			return m, tea.Batch(cmd, tea.Tick(5*time.Second, func(_ time.Time) tea.Msg { return fetchSpotify() }))
		}
		return m, tea.Tick(5*time.Second, func(_ time.Time) tea.Msg { return fetchSpotify() })
	case spotifyImageMsg:
		m.spArt = string(msg)
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	// 44 chars width + 2 padding + 2 border = ~48 cols visual width
	boxWidth := 46

	// 1. Clock
	clockContent := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("TIME"),
		clockStyle.Render(m.time.Format("15:04:05")),
		lipgloss.NewStyle().Foreground(subtle).Render(m.time.Format("Mon, Jan 02")),
	)
	clockBox := boxStyle.Width(boxWidth).Height(6).Render(clockContent)

	// 2. System
	cpuBar := renderBar(m.sys.cpuLoad, 20, warning)
	memBar := renderBar(m.sys.memLoad, 20, highlight)
	sysText := fmt.Sprintf("CPU %2.0f%% %s\nMEM %2.0f%% %s", m.sys.cpuLoad, cpuBar, m.sys.memLoad, memBar)
	if m.sys.gpuLoad >= 0 {
		gpuBar := renderBar(m.sys.gpuLoad, 20, special)
		sysText += fmt.Sprintf("\nGPU %2.0f%% %s", m.sys.gpuLoad, gpuBar)
	}
	sysContent := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("SYSTEM"), sysText)
	sysBox := boxStyle.Width(boxWidth).Height(6).Render(sysContent)

	// 3. Spotify (Vertical High Res)
	spContent := ""
	if m.sp.isPlaying {
		textWidth := 42 // Matches box width minus padding

		info := lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#1DB954")).Bold(true).Width(textWidth).Render(truncate("♪ "+m.sp.track, textWidth)),
			lipgloss.NewStyle().Foreground(subtle).Width(textWidth).Render(truncate(m.sp.artist, textWidth)),
			renderBar(m.sp.progress, textWidth, lipgloss.Color("#1DB954")),
		)

		if m.spArt != "" {
			// Center the art
			centeredArt := lipgloss.NewStyle().Width(textWidth).Align(lipgloss.Center).Render(m.spArt)
			spContent = lipgloss.JoinVertical(lipgloss.Left, centeredArt, " ", info)
		} else {
			spContent = info
		}

		spContent = lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("SPOTIFY"), spContent)
	} else {
		spContent = lipgloss.JoinVertical(lipgloss.Center, titleStyle.Render("SPOTIFY"), "\n\n(Paused)")
	}

	// Height 30 to fit image (22 lines) + text (4 lines) + title (2 lines) + spacing
	spotifyBox := boxStyle.Width(boxWidth).Height(30).Render(spContent)

	// 4. GitHub
	stats := lipgloss.JoinVertical(lipgloss.Left,
		renderStat("Commits", fmt.Sprintf("%d", m.gh.commits)),
		renderStat("PRs Merged", fmt.Sprintf("%d", m.gh.prs)),
		renderStat("Issues", fmt.Sprintf("%d", m.gh.issues)),
	)
	latest := "\nNo recent activity."
	if len(m.gh.latest) > 0 {
		latest = "\nLATEST:\n" + strings.Join(m.gh.latest, "\n")
	}
	ghContent := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("GITHUB"), stats, lipgloss.NewStyle().Foreground(subtle).Render(latest))

	// GitHub height must match total of right column (6+2 + 6+2 + 30+2 = 48ish)
	githubBox := boxStyle.Width(45).Height(46).Render(ghContent)

	rightCol := lipgloss.JoinVertical(lipgloss.Left, clockBox, sysBox, spotifyBox)
	ui := lipgloss.JoinHorizontal(lipgloss.Top, githubBox, rightCol)

	return docStyle.Render(ui)
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}
