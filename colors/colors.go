package colors

import (
	"fmt"
	"runtime"
)

// ANSI color codes
const (
	Reset       = "\033[0m"
	RedColor    = "\033[31m"
	GreenColor  = "\033[32m"
	YellowColor = "\033[33m"
	BlueColor   = "\033[34m"
	PurpleColor = "\033[35m"
	CyanColor   = "\033[36m"
	GreyColor   = "\033[37m"
	WhiteColor  = "\033[97m"

	// Bright colors
	BrightRedColor    = "\033[91m"
	BrightGreenColor  = "\033[92m"
	BrightYellowColor = "\033[93m"
	BrightBlueColor   = "\033[94m"
	BrightPurpleColor = "\033[95m"
	BrightCyanColor   = "\033[96m"

	// Background colors
	BgRedColor   = "\033[41m"
	BgGreenColor = "\033[42m"
	BgBlueColor  = "\033[44m"

	// Text styles
	BoldColor      = "\033[1m"
	DimColor       = "\033[2m"
	ItalicColor    = "\033[3m"
	UnderlineColor = "\033[4m"
)

// colorsEnabled determines if colors should be used
var colorsEnabled = true

// init checks if colors should be disabled on Windows or when output is redirected
func init() {
	// Disable colors on Windows by default (unless explicitly enabled)
	if runtime.GOOS == "windows" {
		colorsEnabled = false
	}
}

// SetColorsEnabled allows enabling or disabling colors
func SetColorsEnabled(enabled bool) {
	colorsEnabled = enabled
}

// IsColorsEnabled returns whether colors are currently enabled
func IsColorsEnabled() bool {
	return colorsEnabled
}

// colorize wraps text with color codes if colors are enabled
func colorize(color, text string) string {
	if !colorsEnabled {
		return text
	}
	return color + text + Reset
}

// Red returns red colored text
func Red(text string) string {
	return colorize(RedColor, text)
}
func Green(text string) string {
	return colorize(GreenColor, text)
}
func Yellow(text string) string {
	return colorize(YellowColor, text)
}
func Blue(text string) string {
	return colorize(BlueColor, text)
}
func Purple(text string) string {
	return colorize(PurpleColor, text)
}
func Cyan(text string) string {
	return colorize(CyanColor, text)
}
func Grey(text string) string {
	return colorize(GreyColor, text)
}
func White(text string) string {
	return colorize(WhiteColor, text)
}
func BrightRed(text string) string {
	return colorize(BrightRedColor, text)
}
func BrightGreen(text string) string {
	return colorize(BrightGreenColor, text)
}
func BrightYellow(text string) string {
	return colorize(BrightYellowColor, text)
}
func BrightBlue(text string) string {
	return colorize(BrightBlueColor, text)
}
func BrightPurple(text string) string {
	return colorize(BrightPurpleColor, text)
}
func BrightCyan(text string) string {
	return colorize(BrightCyanColor, text)
}
func Bold(text string) string {
	return colorize(BoldColor, text)
}
func Dim(text string) string {
	return colorize(DimColor, text)
}
func Italic(text string) string {
	return colorize(ItalicColor, text)
}
func Underline(text string) string {
	return colorize(UnderlineColor, text)
}

// Success returns text in success color (green)
func Success(text string) string {
	return Green(text)
}

// Error returns text in error color (red)
func Error(text string) string {
	return Red(text)
}

// Warning returns text in warning color (yellow)
func Warning(text string) string {
	return Yellow(text)
}

// Info returns text in info color (blue)
func Info(text string) string {
	return Blue(text)
}

// Debug returns text in debug color (grey)
func Debug(text string) string {
	return Grey(text)
}

// Progress returns text in progress color (cyan)
func Progress(text string) string {
	return Cyan(text)
}

// Fetched returns text in fetched color (grey)
func Fetched(text string) string {
	return Grey(text)
}

// Downloaded returns text in download color (blue)
func Downloaded(text string) string {
	return Blue(text)
}

// Saved returns text in saved color (green)
func Saved(text string) string {
	return Green(text)
}

// Printf prints formatted text with color
func Printf(color, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	fmt.Print(colorize(color, text))
}

// Println prints text with color and newline
func Println(color, text string) {
	fmt.Println(colorize(color, text))
}

// FetchedPrintf prints fetched message with formatting
func FetchedPrintf(format string, args ...interface{}) {
	Printf(GreyColor, format, args...)
}
func DownloadedPrintf(format string, args ...interface{}) {
	Printf(BlueColor, format, args...)
}
func SavedPrintf(format string, args ...interface{}) {
	Printf(GreenColor, format, args...)
}
func ErrorPrintf(format string, args ...interface{}) {
	Printf(RedColor, format, args...)
}
func InfoPrintf(format string, args ...interface{}) {
	Printf(BlueColor, format, args...)
}
func WarningPrintf(format string, args ...interface{}) {
	Printf(YellowColor, format, args...)
}
func DebugPrintf(format string, args ...interface{}) {
	Printf(GreyColor, format, args...)
}
