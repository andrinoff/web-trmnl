package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type githubMsg struct {
	commits int
	prs     int
	issues  int
	latest  []string
	status  string
}

func fetchGithubCmd() tea.Cmd {
	return func() tea.Msg { return fetchGithub() }
}

func fetchGithub() tea.Msg {
	username := os.Getenv("GITHUB_USERNAME")
	token := os.Getenv("GITHUB_TOKEN")

	if username == "" {
		return githubMsg{status: "No User"}
	}

	// Fetch events
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/users/%s/events?per_page=100", username), nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return githubMsg{status: "Net Err"}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return githubMsg{status: fmt.Sprintf("Err %d", resp.StatusCode)}
	}

	// Data structures
	type Commit struct {
		Message string `json:"message"`
	}
	type Payload struct {
		Action      string `json:"action"`
		Size        int    `json:"size"`
		PullRequest struct {
			Merged bool `json:"merged"`
		} `json:"pull_request"`
		Commits []Commit `json:"commits"`
	}
	type Event struct {
		Type      string                `json:"type"`
		CreatedAt time.Time             `json:"created_at"`
		Repo      struct{ Name string } `json:"repo"`
		Payload   Payload               `json:"payload"`
	}

	var events []Event
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return githubMsg{status: "Json Err"}
	}

	// Stats Calculation
	todayLocal := time.Now().Format("2006-01-02")
	c, p, i := 0, 0, 0
	var latest []string

	for _, e := range events {
		// Convert event UTC time to Local System Time
		eventLocalTime := e.CreatedAt.In(time.Local)
		dateStr := eventLocalTime.Format("2006-01-02")

		// 1. Latest Activity Logic
		// We want to list individual commits, not just "Push Event"
		if e.Type == "PushEvent" && len(latest) < 4 {
			// Use FULL Repo Name (e.g. "owner/repo")
			rName := e.Repo.Name

			commits := e.Payload.Commits
			if commits != nil && len(commits) > 0 {
				// Iterate backwards to get the most recent commit in the batch first
				for k := len(commits) - 1; k >= 0; k-- {
					if len(latest) >= 4 {
						break
					}

					// Clean up message (take first line, truncate)
					msg := strings.Split(commits[k].Message, "\n")[0]
					if len(msg) > 30 {
						msg = msg[:27] + "..."
					}

					latest = append(latest, fmt.Sprintf("• %s: %s", rName, msg))
				}
			}
		}

		// 2. Today's Stats Logic
		if dateStr != todayLocal {
			continue
		}

		switch e.Type {
		case "PushEvent":
			// STRICT COUNT: Use the payload size (number of commits in the push).
			// If size is missing but commits array exists, use that length.
			count := e.Payload.Size
			if count == 0 && len(e.Payload.Commits) > 0 {
				count = len(e.Payload.Commits)
			}
			c += count

		case "PullRequestEvent":
			if e.Payload.Action == "closed" && e.Payload.PullRequest.Merged {
				p++
			}
		case "IssuesEvent":
			if e.Payload.Action == "closed" {
				i++
			}
		}
	}

	return githubMsg{commits: c, prs: p, issues: i, latest: latest}
}
