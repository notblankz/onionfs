package ui

import (
	"fmt"
	"os"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

func Info(format string, args ...any) {
	fmt.Fprintf(os.Stdout, colorGreen+"[INFO] "+colorReset+format, args...)
}

func Error(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorRed+"[ERROR] "+colorReset+format, args...)
}

func Fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorRed+"[FATAL] "+colorReset+format, args...)
	os.Exit(1)
}

func Warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorYellow+"[WARN]  "+colorReset+format, args...)
}
