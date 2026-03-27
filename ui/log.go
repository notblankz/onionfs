package ui

import (
	"fmt"
	"os"
)

const (
	colorReset     = "\033[0m"
	colorBoldRed   = "\033[1;31m"
	colorBoldGreen = "\033[1;32m"
	colorYellow    = "\033[33m"
	colorBlue      = "\033[34m"
)

func Info(scope string, format string, args ...any) {
	fmt.Fprintf(os.Stdout, colorBoldGreen+"[INFO] "+colorBlue+scope+" "+colorReset+format+"\n", args...)
}

func Error(scope string, format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorBoldRed+"[ERROR] "+colorYellow+scope+" "+colorReset+format+"\n", args...)
}

func Fatal(scope string, format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorBoldRed+"[FATAL] "+colorBlue+scope+" "+colorReset+format+"\n", args...)
	os.Exit(1)
}
