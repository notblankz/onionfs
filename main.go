package main

import (
	"onionfs/ui"
	"os"

	flag "github.com/spf13/pflag"
)

func main() {

	flag.Bool("version", false, "print version and exit")
	// override default usage with custom defined help function
	flag.Usage = ui.PrintHelp

	flag.Parse()

	version, _ := flag.CommandLine.GetBool("version")
	if version {
		ui.PrintVersion()
		os.Exit(0)
	}

}
