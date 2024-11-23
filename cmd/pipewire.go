package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	SIGRTMIN = 34
)

type Pipewire struct {
	Text       string `json:"text"`
	Alt        string `json:"alt"`
	Class      string `json:"class"`
	Tooltip    string `json:"tooltip"`
	Percentage int    `json:"percentage"`
}

var pipewireCmd = &cobra.Command{
	Use:   "pipewire",
	Short: "Pipewire module for waybar",
	Run: func(cmd *cobra.Command, args []string) {
		init := viper.GetBool("init")
		mute := viper.GetBool("mute")
		up := viper.GetInt("up")
		down := viper.GetInt("down")

		switch {
		case init:
			fmt.Println("Put the following object in your waybar config.")
			fmt.Print(`
"custom/pipewire": {
	"interval": 1,
	"signal": 4,
	"return-type": "json",
	"format": "{icon} {percentage}%",
	"format-icons": {
		"normal": ["", "", ""],
		"muted": [" "],
	},
	"exec-if": "which ewmod",
	"exec": "ewmod pipewire",
	"on-click": "ewmod pipewire --mute",
	"on-scroll-up": "ewmod pipewire --up 2",
	"on-scroll-down": "ewmod pipewire --down 2",
},
`)
		case mute:
			err := RunCommand("wpctl", "set-mute", "@DEFAULT_AUDIO_SINK@", "toggle")
			if err != nil {
				Debug("pipewire", "Error toggling mute:", err)
				os.Exit(1)
			}

			err = UpdateWaybar()
			if err != nil {
				Debug("pipewire", "Error updating waybar:", err)
				os.Exit(1)
			}
		case up > 0:
			err := RunCommand("wpctl", "set-volume", "@DEFAULT_AUDIO_SINK@", fmt.Sprintf("%d%%+", up))
			if err != nil {
				Debug("pipewire", "Error setting volume:", err)
				os.Exit(1)
			}
		case down > 0:
			err := RunCommand("wpctl", "set-volume", "@DEFAULT_AUDIO_SINK@", fmt.Sprintf("%d%%-", down))
			if err != nil {
				Debug("pipewire", "Error setting volume:", err)
				os.Exit(1)
			}
		default:
			vol, err := Output("wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@")
			if err != nil {
				Debug("pipewire", "Error getting output:", err)
				os.Exit(1)
			}

			volFields := strings.Fields(vol)

			percentage, err := strconv.ParseFloat(volFields[1], 64)
			if err != nil {
				Debug("pipewire", "Error converting string to float:", err)
				os.Exit(1)
			}
			percentage = percentage * 100

			class := "normal"
			if len(volFields) >= 3 && volFields[2] == "[MUTED]" {
				class = "muted"
			}

			pipewire := Pipewire{
				Text:       fmt.Sprintf("%d", int(percentage)),
				Alt:        class,
				Class:      class,
				Tooltip:    vol,
				Percentage: int(percentage),
			}

			json.NewEncoder(os.Stdout).Encode(pipewire)
		}
	},
}

func init() {
	rootCmd.AddCommand(pipewireCmd)

	pipewireCmd.Flags().SortFlags = false

	pipewireCmd.Flags().Bool("init", false, "Print json code to initialize this module to waybar")
	pipewireCmd.Flags().Bool("mute", false, "Mute pipewire volume")
	pipewireCmd.Flags().Int("up", 0, "Increase pipewire volume")
	pipewireCmd.Flags().Int("down", 0, "Decrease pipewire volume")
	pipewireCmd.MarkFlagsMutuallyExclusive("init", "mute", "up", "down")

	pipewireCmd.Flags().VisitAll(func(flag *pflag.Flag) {
		viper.BindPFlag(flag.Name, flag)
	})
}
