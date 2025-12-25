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

	// Modules
	sys systemMsg
	gh  githubMsg
	sp  spotifyMsg
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

	// Module Updates & Refresh Rates
	case systemMsg:
		m.sys = msg
		return m, tea.Tick(2*time.Second, func(_ time.Time) tea.Msg { return fetchSystem() })
	case githubMsg:
		m.gh = msg
		return m, tea.Tick(60*time.Second, func(_ time.Time) tea.Msg { return fetchGithub() })
	case spotifyMsg:
		m.sp = msg
		return m, tea.Tick(5*time.Second, func(_ time.Time) tea.Msg { return fetchSpotify() })
	}
	return m, nil
}

func (m model) View() string {
	// 1. Clock Card
	clockContent := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("TIME"),
		clockStyle.Render(m.time.Format("15:04:05")),
		lipgloss.NewStyle().Foreground(subtle).Render(m.time.Format("Mon, Jan 02")),
	)
	clockBox := boxStyle.Width(35).Height(6).Render(clockContent)

	// 2. System Card
	cpuBar := renderBar(m.sys.cpuLoad, 15, warning)
	memBar := renderBar(m.sys.memLoad, 15, highlight)

	sysText := fmt.Sprintf("CPU %2.0f%% %s\nMEM %2.0f%% %s", m.sys.cpuLoad, cpuBar, m.sys.memLoad, memBar)
	if m.sys.gpuLoad >= 0 {
		gpuBar := renderBar(m.sys.gpuLoad, 15, special)
		sysText += fmt.Sprintf("\nGPU %2.0f%% %s", m.sys.gpuLoad, gpuBar)
	}

	sysContent := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("SYSTEM"), sysText)
	sysBox := boxStyle.Width(35).Height(6).Render(sysContent)

	// 3. Spotify Card
	spContent := ""
	if m.sp.isPlaying {
		spContent = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("SPOTIFY"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#1DB954")).Bold(true).Render("♪ "+m.sp.track),
			lipgloss.NewStyle().Foreground(subtle).Render(m.sp.artist),
			renderBar(m.sp.progress, 28, lipgloss.Color("#1DB954")),
		)
	} else {
		spContent = lipgloss.JoinVertical(lipgloss.Center, titleStyle.Render("SPOTIFY"), "\n(Paused)")
	}
	spotifyBox := boxStyle.Width(35).Height(8).Render(spContent)

	// 4. GitHub Card
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
	githubBox := boxStyle.Width(45).Height(22).Render(ghContent)

	// Layout Composition
	// Left Col: GitHub
	// Right Col: Clock -> System -> Spotify
	rightCol := lipgloss.JoinVertical(lipgloss.Left, clockBox, sysBox, spotifyBox)
	ui := lipgloss.JoinHorizontal(lipgloss.Top, githubBox, rightCol)

	return docStyle.Render(ui)
}
