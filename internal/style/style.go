// Package style provides consistent terminal styling using Lipgloss.
package style

import "github.com/charmbracelet/lipgloss"

var (
	// Success style for positive outcomes
	Success = lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")). // Green
		Bold(true)

	// Warning style for cautionary messages
	Warning = lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")). // Yellow
		Bold(true)

	// Error style for failures
	Error = lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")). // Red
		Bold(true)

	// Info style for informational messages
	Info = lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")) // Blue

	// Dim style for secondary information
	Dim = lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")) // Gray

	// Bold style for emphasis
	Bold = lipgloss.NewStyle().
		Bold(true)

	// SuccessPrefix is the checkmark prefix for success messages
	SuccessPrefix = Success.Render("✓")

	// WarningPrefix is the warning prefix
	WarningPrefix = Warning.Render("⚠")

	// ErrorPrefix is the error prefix
	ErrorPrefix = Error.Render("✗")

	// ArrowPrefix for action indicators
	ArrowPrefix = Info.Render("→")
)

// RenderSuccess formats a success message
func RenderSuccess(msg string) string {
	return SuccessPrefix + " " + msg
}

// RenderWarning formats a warning message
func RenderWarning(msg string) string {
	return WarningPrefix + " " + msg
}

// RenderError formats an error message
func RenderError(msg string) string {
	return ErrorPrefix + " " + msg
}

// RenderInfo formats an info message with arrow
func RenderInfo(msg string) string {
	return ArrowPrefix + " " + msg
}
