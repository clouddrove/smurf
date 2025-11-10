package terraform

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
)

// These ensure consistent styling across all Terraform commands.
func RedText(text string) string    { return pterm.FgRed.Sprint(text) }
func GreenText(text string) string  { return pterm.FgGreen.Sprint(text) }
func YellowText(text string) string { return pterm.FgYellow.Sprint(text) }
func CyanText(text string) string   { return pterm.FgLightCyan.Sprint(text) }
func GreyText(text string) string   { return pterm.FgGray.Sprint(text) }

// logTime returns the current time formatted as HH:MM:SS
func logTime() string {
	return time.Now().Format("15:04:05")
}

// Info prints informational messages in cyan
func Info(message string, args ...interface{}) {
	timestamp := pterm.LightCyan(logTime())
	text := fmt.Sprintf(message, args...)
	fmt.Printf("%s ℹ %s\n", timestamp, text)
}

// Success prints success messages in green
func Success(message string, args ...interface{}) {
	timestamp := pterm.LightGreen(logTime())
	text := fmt.Sprintf(message, args...)
	coloredText := pterm.LightGreen(text)
	fmt.Printf("%s ✔ %s\n", timestamp, coloredText)
}

// Warn prints warning messages in yellow
func Warn(message string, args ...interface{}) {
	timestamp := pterm.Yellow(logTime())
	text := fmt.Sprintf(message, args...)
	coloredText := pterm.Yellow(text)
	fmt.Printf("%s ⚠ %s\n", timestamp, coloredText)
}

// Error prints error messages in red
func Error(message string, args ...interface{}) {
	timestamp := pterm.LightRed(logTime())
	text := fmt.Sprintf(message, args...)
	coloredText := pterm.LightRed(text)
	fmt.Printf("%s ✖ %s\n", timestamp, coloredText)
}

// Step prints step or progress messages in blue
func Step(message string, args ...interface{}) {
	timestamp := pterm.LightBlue(logTime())
	text := fmt.Sprintf(message, args...)
	coloredText := pterm.LightBlue(text)
	fmt.Printf("%s ▶ %s\n", timestamp, coloredText)
}
