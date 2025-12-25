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

	// 1. Fetch Events
	url := fmt.Sprintf("https://api.github.com/users/%s/events?per_page=100", username)
	req, _ := http.NewRequest("GET", url, nil)
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

	// Data Structures
	type Commit struct {
		Message string `json:"message"`
	}
	type Payload struct {
		Action      string `json:"action"`
		Size        int    `json:"size"`
		Ref         string `json:"ref"`
		Head        string `json:"head"`
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

	// 2. Process
	todayLocal := time.Now().Format("2006-01-02")
	c, p, i := 0, 0, 0
	var latest []string

	for _, e := range events {
		dateStr := e.CreatedAt.In(time.Local).Format("2006-01-02")

		// --- STRICT FILTER: ONLY TODAY ---
		if dateStr != todayLocal {
			continue
		}

		// --- Latest Activity List (Push Only) ---
		if e.Type == "PushEvent" && len(latest) < 4 {
			rName := e.Repo.Name

			// Case A: Payload has commits
			if len(e.Payload.Commits) > 0 {
				for k := len(e.Payload.Commits) - 1; k >= 0; k-- {
					if len(latest) >= 4 {
						break
					}
					msg := formatMessage(e.Payload.Commits[k].Message)
					latest = append(latest, fmt.Sprintf("• %s: %s", rName, msg))
				}
			} else {
				// Case B: Payload empty (Branch update/Merge), fetch manually
				msg := "Pushed update"
				if e.Payload.Head != "" && token != "" {
					fetchedMsg := fetchSingleCommitMessage(e.Repo.Name, e.Payload.Head, token)
					if fetchedMsg != "" {
						msg = fetchedMsg
					} else if e.Payload.Ref != "" {
						branch := strings.Replace(e.Payload.Ref, "refs/heads/", "", 1)
						msg = fmt.Sprintf("Pushed to %s", branch)
					}
				}
				latest = append(latest, fmt.Sprintf("• %s: %s", rName, msg))
			}
		}

		// --- Stats Counters ---
		switch e.Type {
		case "PushEvent":
			count := e.Payload.Size
			if count == 0 {
				count = 1
			} // Force count 1
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

// Helper to clean up commit messages
func formatMessage(raw string) string {
	lines := strings.Split(raw, "\n")
	msg := lines[0]
	if len(msg) > 30 {
		msg = msg[:27] + "..."
	}
	return msg
}

// Helper to fetch a specific commit message
func fetchSingleCommitMessage(repoFull string, sha string, token string) string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", repoFull, sha)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	var commitData struct {
		Commit struct {
			Message string `json:"message"`
		} `json:"commit"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&commitData); err != nil {
		return ""
	}

	return formatMessage(commitData.Commit.Message)
}
