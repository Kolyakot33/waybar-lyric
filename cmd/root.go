package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	Version = "Git"
)

var rootCmd = &cobra.Command{Use: "ewmod", Version: Version, Short: "Ephemeral's waybar modules"}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func Debug(scope string, a ...any) {
	fmt.Fprintf(os.Stderr, "[%s] ", scope)
	fmt.Fprintln(os.Stderr, a...)
}

func RunCommand(bin string, args ...string) error {
	fmt.Fprintf(os.Stderr, "[sys] running: %s ", bin)
	for _, a := range args {
		fmt.Fprintf(os.Stderr, "%q ", a)
	}
	fmt.Print("\n")

	cmd := exec.Command(bin, args...)

	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}

func Output(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(output), "\n"), nil
}

func UpdateWaybar() error {
	return SendSignal("^waybar$", SIGRTMIN+4)
}

func SendSignal(processName string, signal int) error {
	cmd := exec.Command("pgrep", processName)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find processes matching %q: %w", processName, err)
	}

	pidStrings := strings.Fields(string(output))
	if len(pidStrings) == 0 {
		return fmt.Errorf("no processes found matching %q", processName)
	}

	for _, pidStr := range pidStrings {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			return fmt.Errorf("invalid PID %q: %w", pidStr, err)
		}

		// Send the signal
		err = syscall.Kill(pid, syscall.Signal(signal))
		if err != nil {
			return fmt.Errorf("failed to send signal to PID %d: %w", pid, err)
		}
	}

	return nil
}
