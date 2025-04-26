package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Nadim147c/go-mpris"
	"github.com/godbus/dbus/v5"
)

const (
	SleepTime = 500 * time.Millisecond
	Version   = "waybar-lyric v0.8.0 (https://github.com/Nadim147c/waybar-lyric)"
)

func truncate(input string) string {
	r := []rune(input)

	if len(r) <= MaxTextLength {
		return input
	}

	if MaxTextLength > 3 {
		return string(r[:MaxTextLength-3]) + "..."
	}

	return string(r[:MaxTextLength])
}

func main() {
	if TooltipLines < 4 {
		fmt.Fprintln(os.Stderr, "Tooltip lines limit must be at least 4")
		return
	}

	if PrintVersion {
		fmt.Fprint(os.Stderr, Version)
		return
	}

	if PrintInit {
		PrintSnippet()
		return
	}

	conn, err := dbus.SessionBus()
	if err != nil {
		slog.Error("Failed to create dbus connection", "error", err)
		return
	}

	// Call ListNames on org.freedesktop.DBus
	var names []string
	err = conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus").
		Call("org.freedesktop.DBus.ListNames", 0).
		Store(&names)
	if err != nil {
		log.Fatal(err)
	}

	// Filter MPRIS players
	var playerName string
	for _, name := range names {
		if len(name) >= len("org.mpris.MediaPlayer2.") && name[:len("org.mpris.MediaPlayer2.")] == "org.mpris.MediaPlayer2." {
			playerName = name
		}
	}

	player := mpris.New(conn, playerName)

	if ToggleState {
		slog.Info("Toggling player state")
		if err := player.PlayPause(); err != nil {
			slog.Error("Failed to toggle player state", "error", err)
		}
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	psChan := make(chan *dbus.Signal, 0)
	player.OnSignal(psChan)

	lyricTicker := time.NewTicker(SleepTime)
	defer lyricTicker.Stop()

	// Main loop
	fixedTicker := time.NewTicker(SleepTime)
	defer fixedTicker.Stop()

	var lastInfo *PlayerInfo = nil
	var lastLine *LyricLine = nil
	var lyricsNotFound bool

	playerOpened := true

	for {
		select {
		case <-ctx.Done():
			return // Clean exit on cancel
		case <-psChan:
			slog.Debug("Received player update signal")
		case <-lyricTicker.C:
		case <-fixedTicker.C:
		}

		if _, err := player.GetPosition(); err != nil {
			if playerOpened {
				slog.Error("Player not found!", "error", err)
				fmt.Println("{}")
				playerOpened = false
			}
			continue
		} else {
			playerOpened = true
		}

		info, err := GetSpotifyInfo(player)
		if err != nil {
			slog.Error("Failed to parse dbus mpris metadata", "error", err)
			fmt.Println("{}")
			continue
		}

		playerUpdated := lastInfo == nil || lastInfo.ID != info.ID || lastInfo.Status != info.Status

		if playerUpdated {
			slog.Info("Player media found", "title", info.Title, "artist", info.Artist, "status", info.Status)
			lastInfo = info
		}

		if info.Status == mpris.PlaybackStopped {
			slog.Info("Player is stopped")
			fmt.Println("{}")
			continue
		}

		if info.Status == mpris.PlaybackPaused {
			if playerUpdated {
				info.Waybar().Encode()
				lastLine = nil
			}
			continue
		}

		lyrics, err := GetLyrics(info)
		if err != nil {
			if !lyricsNotFound {
				slog.Error("Failed to get lyrics", "error", err)
				info.Waybar().Encode()
				lyricsNotFound = true
			}
			continue
		}
		lyricsNotFound = false

		idx := -1
		for i, line := range lyrics {
			if info.Position <= line.Timestamp {
				break
			}
			idx = i
		}

		if idx == -1 {
			if lastLine != nil && lastLine.Timestamp == -1 {
				continue
			}
			lastLine = &LyricLine{Timestamp: -1, Text: ""}

			var tooltip strings.Builder
			tooltip.WriteString("<b><big>󰝚 </big></b>\n")
			tooltip.WriteString(fmt.Sprintf("<span foreground=\"%s\">", TootlipColor))

			end := min(TooltipLines, len(lyrics))
			tooltipLyrics := lyrics[:end]
			for _, ttl := range tooltipLyrics {
				if ttl.Text != "" {
					tooltip.WriteString(ttl.Text + "\n")
				} else {
					tooltip.WriteString("󰝚 \n")
				}
			}

			waybar := info.Waybar()
			waybar.Tooltip = strings.TrimSpace(tooltip.String()) + "</span>"
			waybar.Alt = Music
			waybar.Class = Class{Playing, Music}
			waybar.Encode()
		} else {
			lyric := lyrics[idx]
			if lastLine != nil && lastLine.Timestamp == lyric.Timestamp {
				continue
			}
			lastLine = &lyric

			slog.Info("Lyrics", "line", lyric.Text)

			waybar := NewWaybar(lyrics, idx, info.Percentage())
			if lyric.Text != "" {
				waybar.Encode()
			} else {
				waybar.Text = fmt.Sprintf("%s - %s", info.Artist, info.Title)
				waybar.Alt = Music
				waybar.Encode()
			}

			if len(lyrics) > idx+1 {
				n := lyrics[idx+1]
				d := n.Timestamp - info.Position
				slog.Debug("Sleep", "duration", d.String(), "position", info.Position.String(), "next", n.Timestamp.String())
				lyricTicker.Reset(d)
			}
		}

	}
}
