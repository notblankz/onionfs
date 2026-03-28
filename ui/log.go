package ui

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

const (
	colorReset     = "\033[0m"
	colorBoldRed   = "\033[1;31m"
	colorBoldGreen = "\033[1;32m"
	colorYellow    = "\033[33m"
	colorBlue      = "\033[34m"
)

var useColor = term.IsTerminal(int(os.Stdout.Fd()))

func SetNoColor() {
	useColor = false
}

func buildPrefix(level string, levelColor string, scope string, scopeColor string) string {
	if useColor {
		return levelColor + "[" + level + "] " + colorReset +
			scopeColor + scope + " " + colorReset
	}
	return "[" + level + "] " + scope + " "
}

func Info(scope string, format string, args ...any) {
	prefix := buildPrefix("INFO", colorBoldGreen, scope, colorBlue)
	fmt.Fprintf(os.Stdout, prefix+format+"\n", args...)
}

func Error(scope string, format string, args ...any) {
	prefix := buildPrefix("ERROR", colorBoldRed, scope, colorYellow)
	fmt.Fprintf(os.Stderr, prefix+format+"\n", args...)
}

func Fatal(scope string, format string, args ...any) {
	prefix := buildPrefix("FATAL", colorBoldRed, scope, colorBlue)
	fmt.Fprintf(os.Stderr, prefix+format+"\n", args...)
	os.Exit(1)
}
