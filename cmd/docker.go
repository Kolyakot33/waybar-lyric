package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type (
	Docker struct {
		Text    string `json:"text"`
		Alt     string `json:"alt"`
		Class   string `json:"class"`
		Tooltip string `json:"tooltip"`
	}

	ContainerStats struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		CPUPerc  string `json:"CPUPerc"`
		MemUsage string `json:"MemUsage"`
		MemPerc  string `json:"MemPerc"`
		NetIO    string `json:"NetIO"`
		BlockIO  string `json:"BlockIO"`
		PIDs     string `json:"PIDs"`
	}
)

func getDockerStats() ([]ContainerStats, error) {
	cmd := exec.Command("docker", "stats", "--no-stream", "--format", "json")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("error creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("error starting command: %w", err)
	}

	var stats []ContainerStats
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		var stat ContainerStats
		if err := json.Unmarshal(scanner.Bytes(), &stat); err != nil {
			return nil, fmt.Errorf("error parsing JSON line: %w", err)
		}
		stats = append(stats, stat)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading command output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("error waiting for command to finish: %w", err)
	}

	return stats, nil
}

func generateSimpleTable(headers []string, rows [][]string) string {
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, col := range row {
			if len(col) > colWidths[i] {
				colWidths[i] = len(col)
			}
		}
	}

	var builder strings.Builder

	for i, header := range headers {
		fmt.Fprintf(&builder, "%-*s", colWidths[i]+2, header)
	}
	builder.WriteString("\n")

	for _, width := range colWidths {
		builder.WriteString(strings.Repeat("-", width+2))
	}
	builder.WriteString("\n")

	for _, row := range rows {
		for i, col := range row {
			fmt.Fprintf(&builder, "%-*s", colWidths[i]+2, col)
		}
		builder.WriteString("\n")
	}

	return strings.TrimSpace(builder.String())
}

func toTitle(input string) string {
	input = strings.ReplaceAll(input, "-", " ")
	input = strings.ReplaceAll(input, "_", " ")

	return cases.Title(language.English).String(input)
}

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Docker modules for waybar",
	RunE: func(cmd *cobra.Command, args []string) error {
		viper.BindPFlag("init", cmd.Flags().Lookup("init"))
		Log = func(a ...any) {
			WriteLog("Docker", a...)
		}

		if viper.GetBool("init") {
			fmt.Print(`Put the following object in your waybar config:

"custom/docker": {
	"interval": 5,
	"signal": 4,
	"return-type": "json",
	"format": "{icon} {0}",
	"format-icons": [""],
	"exec-if": "! docker ps --quiet",
	"exec": "waytune docker",
},
`)
			os.Exit(0)
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetEscapeHTML(false)

		stats, err := getDockerStats()
		if err != nil {
			panic(err)
		}

		var tableRows [][]string
		for _, stat := range stats {
			name := toTitle(stat.Name)
			tableRows = append(tableRows, []string{name, stat.CPUPerc, stat.MemPerc})
		}

		tooltip := generateSimpleTable([]string{"Name", "CPU", "MEM"}, tableRows)

		encoder.Encode(Docker{
			Text:    fmt.Sprintf("%d", len(stats)),
			Alt:     "docker",
			Class:   "docker",
			Tooltip: tooltip,
		})

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dockerCmd)

	rootCmd.AddCommand(dockerCmd)

	dockerCmd.Flags().Bool("init", false, "Print json code to initialize this module to waybar")

	dockerCmd.Flags().VisitAll(func(f *pflag.Flag) {
		viper.GetViper().BindPFlag(f.Name, f)
	})
}
