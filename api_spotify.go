package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nfnt/resize"
)

type spotifyMsg struct {
	isPlaying bool
	track     string
	artist    string
	progress  float64
	imageUrl  string
	status    string
}

type spotifyImageMsg string

func fetchSpotifyCmd() tea.Cmd {
	return func() tea.Msg { return fetchSpotify() }
}

func downloadImageCmd(url string) tea.Cmd {
	return func() tea.Msg { return downloadSpotifyImage(url) }
}

func fetchSpotify() tea.Msg {
	id := os.Getenv("SPOTIFY_CLIENT_ID")
	secret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	refresh := os.Getenv("SPOTIFY_REFRESH_TOKEN")

	if refresh == "" {
		return spotifyMsg{status: "No Token"}
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refresh)

	req, _ := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(id+":"+secret)))

	client := &http.Client{Timeout: 5 * time.Second}
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
	json.NewDecoder(resp.Body).Decode(&tokenResp)

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
			Album      struct {
				Images []struct{ Url string } `json:"images"`
			} `json:"album"`
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

	imgUrl := ""
	if len(playResp.Item.Album.Images) > 0 {
		// CHANGE: Use Index 1 (300x300) instead of last (64x64) for better quality source
		idx := 1
		if idx >= len(playResp.Item.Album.Images) {
			idx = len(playResp.Item.Album.Images) - 1
		}
		imgUrl = playResp.Item.Album.Images[idx].Url
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
		imageUrl:  imgUrl,
	}
}

func downloadSpotifyImage(urlStr string) tea.Msg {
	if urlStr == "" {
		return spotifyImageMsg("")
	}

	resp, err := http.Get(urlStr)
	if err != nil {
		return spotifyImageMsg("")
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return spotifyImageMsg("")
	}

	// --- HIGH QUALITY RENDER LOGIC ---
	// We want the image to fit in width=40 chars.
	// Since we use half-blocks (2 pixels vertical per char),
	// the internal height resolution is doubled.

	widthChars := uint(40)
	// Aspect ratio correction: terminal cells are usually ~0.5 aspect (tall).
	// To get a square image: Pixel Width = 40. Pixel Height = 40.
	// 40 vertical pixels / 2 pixels per char = 20 chars height.

	internalWidth := widthChars
	internalHeight := widthChars // Square aspect ratio in pixels

	resized := resize.Resize(internalWidth, internalHeight, img, resize.Lanczos3)
	bounds := resized.Bounds()

	var sb strings.Builder

	// Iterate y by 2 since we pack 2 vertical pixels into 1 char
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 2 {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Top Pixel
			r1, g1, b1, _ := resized.At(x, y).RGBA()
			cTop := color.RGBA{uint8(r1 >> 8), uint8(g1 >> 8), uint8(b1 >> 8), 255}

			// Bottom Pixel (check bounds just in case)
			var cBot color.RGBA
			if y+1 < bounds.Max.Y {
				r2, g2, b2, _ := resized.At(x, y+1).RGBA()
				cBot = color.RGBA{uint8(r2 >> 8), uint8(g2 >> 8), uint8(b2 >> 8), 255}
			} else {
				cBot = cTop // Fallback
			}

			// Render Half-Block '▀'
			// Foreground = Top Color, Background = Bottom Color
			style := lipgloss.NewStyle().
				Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", cTop.R, cTop.G, cTop.B))).
				Background(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", cBot.R, cBot.G, cBot.B)))

			sb.WriteString(style.Render("▀"))
		}
		sb.WriteString("\n")
	}

	return spotifyImageMsg(sb.String())
}
