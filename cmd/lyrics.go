package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/Pauloo27/go-mpris"
	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const LRCLIB_ENDPOINT = "https://lrclib.net/api/get"

type (
	Lyrics struct {
		Text       string      `json:"text"`
		Class      interface{} `json:"class"`
		Alt        string      `json:"alt"`
		Tooltip    string      `json:"tooltip"`
		Percentage int         `json:"percentage"`
	}

	LrcLibResponse struct {
		ID           int     `json:"id"`
		Name         string  `json:"name"`
		TrackName    string  `json:"trackName"`
		ArtistName   string  `json:"artistName"`
		AlbumName    string  `json:"albumName"`
		Duration     float64 `json:"duration"`
		Instrumental bool    `json:"instrumental"`
		PlainLyrics  string  `json:"plainLyrics"`
		SyncedLyrics string  `json:"syncedLyrics"`
	}

	LyricLine struct {
		Timestamp float64
		Text      string
	}
)

func fetchLyrics(url string, uri string) ([]LyricLine, error) {
	userCacheDir, _ := os.UserCacheDir()
	cacheDir := filepath.Join(userCacheDir, "EWM-Lyrics")

	uri = strings.ReplaceAll(uri, "/", "-")
	cacheFile := filepath.Join(cacheDir, uri+".csv")

	if cahcedLyrics, err := loadCache(cacheFile); err == nil {
		return cahcedLyrics, nil
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch lyrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var resJson LrcLibResponse
	err = json.Unmarshal(body, &resJson)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	lyrics, err := parseLyrics(resJson.SyncedLyrics)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lyrics: %w", err)
	}

	if len(lyrics) == 0 {
		return nil, fmt.Errorf("failed to find sync lyrics lines")
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	err = saveCache(lyrics, cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to cache lyrics to psudo csv: %w", err)
	}

	return lyrics, nil
}

func parseLyrics(file string) ([]LyricLine, error) {
	var lyrics []LyricLine
	lines := strings.Split(file, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "]", 2)
		if len(parts) != 2 {
			continue
		}
		timestampStr := strings.TrimPrefix(parts[0], "[")
		lyric := strings.TrimSpace(parts[1])

		timestamp, err := parseTimestamp(timestampStr)
		if err != nil {
			continue
		}

		lyrics = append(lyrics, LyricLine{Timestamp: timestamp, Text: lyric})
	}
	return lyrics, nil
}

func parseTimestamp(ts string) (float64, error) {
	parts := strings.Split(ts, ":")

	var seconds float64

	for i := len(parts) - 1; i >= 0; i-- {
		part, err := strconv.ParseFloat(strings.TrimSpace(parts[i]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp part: %s", parts[i])
		}

		seconds += part * math.Pow(60, float64(len(parts)-1-i))
	}

	return seconds, nil
}

func saveCache(lines []LyricLine, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, line := range lines {
		_, err := fmt.Fprintf(file, "%.6f,%s\n", line.Timestamp, line.Text)
		if err != nil {
			return err
		}
	}
	return nil
}

func loadCache(filePath string) ([]LyricLine, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []LyricLine
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ",", 2)
		if len(parts) != 2 {
			continue // Skip invalid lines
		}

		timestamp, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return nil, err
		}

		lines = append(lines, LyricLine{Timestamp: timestamp, Text: parts[1]})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func truncate(input string, limit int) string {
	if len(input) <= limit {
		return input
	}

	if limit > 3 {
		return input[:limit-3] + "..."
	}

	return input[:limit]
}

var lyricsCmd = &cobra.Command{
	Use:   "lyrics",
	Short: "Lyrics modules for ",
	Run: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("init", cmd.Flags().Lookup("init"))

		if viper.GetBool("init") {
			fmt.Println("Put the following object in your waybar config.")
			fmt.Print(`
"custom/lyrics": {
	"interval": 1,
	"signal": 4,
	"return-type": "json",
	"format": "{icon} {0}",
	"format-icons": {
		"playing": "",
		"paused": "",
		"lyric": "",
	},
	"exec-if": "which ewmod",
	"exec": "ewmod lyrics --max-length 100",
	"on-click": "ewmod lyrics --toggle",
},
`)
			os.Exit(0)
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetEscapeHTML(false)

		lockFile := filepath.Join(os.TempDir(), "EWM-Lyrics.lock")
		file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			Debug("Lyrics", "Failed to open or create lock file:", err)
			os.Exit(1)
		}
		defer file.Close()

		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err != nil {
			if err == syscall.EWOULDBLOCK {
				Debug("Lyrics", "Another instance of the CLI is already running. Exiting.")
				os.Exit(0)
			}
			Debug("Lyrics", "Failed to acquire lock:", err)
			os.Exit(1)
		}

		defer os.Remove(lockFile)

		conn, err := dbus.SessionBus()
		if err != nil {
			Debug("Lyrics", err)
			os.Exit(1)
		}
		names, err := mpris.List(conn)
		if err != nil {
			Debug("Lyrics", err)
			os.Exit(1)
		}

		searchTerm := "spotify"
		var playerName string
		for _, name := range names {
			if strings.Contains(strings.ToLower(name), strings.ToLower(searchTerm)) {
				playerName = name
				break
			}
		}

		if playerName == "" {
			Debug("Lyrics", "failed to find player")
		}

		player := mpris.New(conn, playerName)

		if viper.GetBool("toggle") {
			player.PlayPause()
			UpdateWaybar()
			os.Exit(0)
		}

		meta, err := player.GetMetadata()
		if err != nil {
			Debug("Lyrics", err)
			os.Exit(1)
		}

		status, err := player.GetPlaybackStatus()
		if err != nil {
			Debug("Lyrics", err)
			os.Exit(1)
		}

		artist := meta["xesam:artist"].Value().([]string)[0]
		title := meta["xesam:title"].Value().(string)
		album := meta["xesam:album"].Value().(string)
		duration := float64(meta["mpris:length"].Value().(uint64)) / 1000000
		position, _ := player.GetPosition()

		if title == "" || artist == "" {
			os.Exit(1)
		}

		if status == "Paused" {
			encoder.Encode(Lyrics{
				Text:       fmt.Sprintf("%s - %s", artist, title),
				Class:      "info",
				Alt:        "paused",
				Tooltip:    "",
				Percentage: int(100 * position / duration),
			})
			os.Exit(0)
		}

		if status == "Stopped" {
			os.Exit(0)
		}

		queryParams := url.Values{}
		queryParams.Set("track_name", title)
		queryParams.Set("artist_name", artist)
		if album != "" {
			queryParams.Set("album_name", album)
		}
		if duration != 0 {
			queryParams.Set("duration", fmt.Sprintf("%.1f", duration))
		}
		params := queryParams.Encode()

		url := fmt.Sprintf("%s?%s", LRCLIB_ENDPOINT, params)
		uri := filepath.Base(meta["mpris:trackid"].Value().(string))

		lyrics, err := fetchLyrics(url, uri)
		if err != nil {
			Debug("Lyrics", err)
			encoder.Encode(Lyrics{
				Text:       fmt.Sprintf("%s - %s", artist, title),
				Class:      "info",
				Alt:        "playing",
				Percentage: int(100 * position / duration),
			})
			os.Exit(0)
		}

		var idx int
		for i, line := range lyrics {
			if position < line.Timestamp {
				break
			}
			idx = i
		}

		currentLine := lyrics[idx].Text

		if currentLine != "" {
			start := idx - 2
			if start < 0 {
				start = 0
			}

			end := idx + 5
			if end > len(lyrics) {
				end = len(lyrics)
			}

			tooltipLyrics := lyrics[start:end]
			var tooltip strings.Builder
			for i, ttl := range tooltipLyrics {
				lineText := ttl.Text
				if start+i == idx {
					tooltip.WriteString("> ")
				}
				tooltip.WriteString(lineText + "\n")
			}

			encoder.Encode(Lyrics{
				Text:       truncate(currentLine, viper.GetInt("max-length")),
				Class:      "lyric",
				Alt:        "lyric",
				Tooltip:    strings.TrimSpace(tooltip.String()),
				Percentage: int(100 * position / duration),
			})
			os.Exit(0)
		}

		encoder.Encode(Lyrics{
			Text:       fmt.Sprintf("%s - %s", artist, title),
			Class:      "info",
			Alt:        "playing",
			Tooltip:    "",
			Percentage: int(100 * position / duration),
		})
	},
}

func init() {
	rootCmd.AddCommand(lyricsCmd)
	lyricsCmd.Flags().Bool("init", false, "Print json code to initialize this module to waybar")
	lyricsCmd.Flags().Bool("toggle", false, "Play if paused. Pause if playing")
	lyricsCmd.Flags().Int32("max-length", 100, "Truncate lyric line up to specific length")
	lyricsCmd.MarkFlagsMutuallyExclusive("init", "toggle", "max-length")

	lyricsCmd.Flags().VisitAll(func(f *pflag.Flag) {
		viper.BindPFlag(f.Name, f)
	})
}
