package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type spotifyMsg struct {
	isPlaying bool
	track     string
	artist    string
	progress  float64
	status    string
}

func fetchSpotifyCmd() tea.Cmd {
	return func() tea.Msg { return fetchSpotify() }
}

func fetchSpotify() tea.Msg {
	id := os.Getenv("SPOTIFY_CLIENT_ID")
	secret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	refresh := os.Getenv("SPOTIFY_REFRESH_TOKEN")

	if refresh == "" {
		return spotifyMsg{status: "No Token"}
	}

	// 1. Refresh Access Token
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refresh)

	req, _ := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(id+":"+secret)))

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return spotifyMsg{status: "Net Err"}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return spotifyMsg{status: "Auth Err"}
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return spotifyMsg{status: "Json Err"}
	}

	// 2. Get Player Data
	req, _ = http.NewRequest("GET", "https://api.spotify.com/v1/me/player/currently-playing", nil)
	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	resp, err = client.Do(req)
	if err != nil {
		return spotifyMsg{status: "API Err"}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return spotifyMsg{isPlaying: false}
	}

	var playResp struct {
		IsPlaying  bool `json:"is_playing"`
		ProgressMs int  `json:"progress_ms"`
		Item       struct {
			Name       string                  `json:"name"`
			DurationMs int                     `json:"duration_ms"`
			Artists    []struct{ Name string } `json:"artists"`
		} `json:"item"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&playResp); err != nil {
		return spotifyMsg{isPlaying: false}
	}

	if !playResp.IsPlaying {
		return spotifyMsg{isPlaying: false}
	}

	artist := "Unknown"
	if len(playResp.Item.Artists) > 0 {
		artist = playResp.Item.Artists[0].Name
	}

	pct := 0.0
	if playResp.Item.DurationMs > 0 {
		pct = (float64(playResp.ProgressMs) / float64(playResp.Item.DurationMs)) * 100
	}

	return spotifyMsg{
		isPlaying: true,
		track:     playResp.Item.Name,
		artist:    artist,
		progress:  pct,
	}
}
