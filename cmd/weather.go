package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const WEATHER_API_ENDPOINT = "https://api.open-meteo.com/v1/forecast"

type (
	Weather struct {
		Text    string      `json:"text"`
		Class   interface{} `json:"class"`
		Alt     string      `json:"alt"`
		Tooltip string      `json:"tooltip"`
	}

	WeatherResponse struct {
		Current WeatherCurrent `json:"current"`
	}

	WeatherCurrent struct {
		Temperature float64 `json:"temperature_2m"`
		Code        int     `json:"weather_code"`
	}
)

func fetchWeather(url string) (*WeatherResponse, error) {
	cacheFile := filepath.Join(os.TempDir(), "EWM-Weather.json")
	cacheDuration := 5 * time.Minute

	if info, err := os.Stat(cacheFile); err == nil {
		if time.Since(info.ModTime()) < cacheDuration {
			cacheData, err := os.ReadFile(cacheFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read cache file: %w", err)
			}

			var cachedWeather WeatherResponse
			if err := json.Unmarshal(cacheData, &cachedWeather); err != nil {
				return nil, fmt.Errorf("failed to decode cached data: %w", err)
			}
			Log("Loading cached weather")
			return &cachedWeather, nil
		}
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch weather data from API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned non-200 status code: %d", resp.StatusCode)
	}

	var weatherData WeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherData); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	cacheData, err := json.Marshal(weatherData)
	if err != nil {
		return nil, fmt.Errorf("failed to encode weather data for cache: %w", err)
	}

	if err := os.WriteFile(cacheFile, cacheData, 0644); err != nil {
		fmt.Println("Warning: Failed to save data to cache:", err)
	}

	return &weatherData, nil
}

var weatherCmd = &cobra.Command{
	Use:   "weather",
	Short: "Weather module for waybar",
	Run: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("init", cmd.Flags().Lookup("init"))

		Log = func(a ...any) {
			WriteLog("Weather", a...)
		}

		if viper.GetBool("init") {
			fmt.Print(`Create a file with location on your home directory with 'touch ~/.EWM-Weather'
Put current location (latitude,longitude) in '~/.EWM-Weather'

For example: "echo '20.32,60.21' > ~/.EWM-Weather"

Put the following object in your waybar config:

"custom/weather": {
	"interval": 60,
	"signal": 4,
	"return-type": "json",
	"format": "{icon} {0}",
	"format-icons": {
		"clear": "",
		"cloudy": "",
		"fog": "󰖑",
		"drizzle": "",
		"rain": "󰖗",
		"snow": "",
		"thunderstorm": "",
	},
	"exec-if": "which ewmod",
	"exec": "ewmod weather --unit c",
},
`)
			os.Exit(0)
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetEscapeHTML(false)

		home, err := os.UserHomeDir()
		if err != nil {
			Log("Failed to find home path:", err)
			os.Exit(1)
		}

		weatherFilePath := filepath.Join(home, ".EWM-Weather")

		weatherFile, err := os.Open(weatherFilePath)
		if err != nil {
			Log("Failed to open weather info file:", err)
			os.Exit(1)
		}

		defer weatherFile.Close()

		var latitude float64
		var longitude float64

		_, err = fmt.Fscanf(weatherFile, "%f,%f", &latitude, &longitude)
		if err != nil {
			Log("Failed to parse weather info file", err)
			os.Exit(1)
		}

		queryParams := url.Values{}
		queryParams.Set("latitude", fmt.Sprintf("%f", latitude))
		queryParams.Set("longitude", fmt.Sprintf("%f", longitude))
		queryParams.Set("current", "temperature_2m")
		params := queryParams.Encode()

		url := fmt.Sprintf("%s?%s", WEATHER_API_ENDPOINT, params)

		weather, err := fetchWeather(url)
		if err != nil {
			Log("failed to fetch lyrics ", err)
			os.Exit(1)
		}

		weatherCodes := map[int]string{
			0:  "Clear skies",
			1:  "Mainly clear",
			2:  "Partly cloudy",
			3:  "Overcast",
			45: "Fog",
			48: "Depositing rime fog",
			51: "Light drizzle",
			53: "Moderate drizzle",
			55: "Dense drizzle",
			61: "Slight rain",
			63: "Moderate rain",
			65: "Heavy rain",
			71: "Slight snowfall",
			73: "Moderate snowfall",
			75: "Heavy snowfall",
			95: "Slight thunderstorm",
			96: "Thunderstorm with heavy hail",
		}

		var icon string
		switch weather.Current.Code {
		case 0, 1:
			icon = "clear"
		case 2, 3:
			icon = "cloudy"
		case 45, 48:
			icon = "fog"
		case 51, 52, 53:
			icon = "drizzle"
		case 61, 62, 63:
			icon = "rain"
		case 71, 72, 73:
			icon = "snow"
		case 95, 96:
			icon = "thunderstorm"
		}

		condition := weatherCodes[weather.Current.Code]

		var text string
		switch strings.ToLower(viper.GetString("unit")) {
		case "f":
			text = fmt.Sprintf("%.1f°F", weather.Current.Temperature*9/5+32)
		case "k":
			text = fmt.Sprintf("%.1fK", weather.Current.Temperature+273.15)
		default:
			text = fmt.Sprintf("%.1f°C", weather.Current.Temperature)
		}

		encoder.Encode(Weather{
			Text:    text,
			Class:   []string{condition, icon},
			Alt:     icon,
			Tooltip: fmt.Sprintf("%s/%s", text, condition),
		})
	},
}

func init() {
	rootCmd.AddCommand(weatherCmd)

	weatherCmd.Flags().Bool("init", false, "Print json code to initialize this module to waybar")
	weatherCmd.Flags().String("unit", "c", "Specify unit to use for temperature. (choices: c, f, k) ")

	weatherCmd.Flags().VisitAll(func(f *pflag.Flag) {
		viper.GetViper().BindPFlag(f.Name, f)
	})
}
