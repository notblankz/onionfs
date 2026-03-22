package main

import (
	"fmt"
	"onionfs/ui"
	"os"

	flag "github.com/spf13/pflag"
)

func main() {

	var noCow, showMeta, daemon bool

	flag.StringP("lower", "l", "", "lower (read-only) directory [required]")
	flag.StringP("upper", "u", "", "upper (read-write) directory [required]")
	flag.StringP("mountpoint", "m", "", "mount point [required]")
	flag.BoolVar(&noCow, "no-cow", false, "disable copy-on-write")
	flag.BoolVar(&showMeta, "show-meta", false, "show .wh.* files in directory listings")
	flag.BoolVar(&daemon, "daemon", false, "run in background (default: foreground)")
	flag.Bool("version", false, "print version and exit")
	// override default usage with custom defined help function
	flag.Usage = ui.PrintHelp

	flag.Parse()

	lowerDir, _ := flag.CommandLine.GetString("lower")
	upperDir, _ := flag.CommandLine.GetString("upper")
	mountpoint, _ := flag.CommandLine.GetString("mountpoint")

	if len(lowerDir) != 0 && len(upperDir) != 0 && len(mountpoint) != 0 {
		fmt.Println("OnionFS")
		ui.Info("Lower Directory: %s\n", lowerDir)
		ui.Info("Upper Directory: %s\n", upperDir)
		ui.Info("Mountpoint: %s\n", mountpoint)
		os.Exit(0)
	} else {
		ui.Error("Missing required parameters")
		ui.PrintHelp()
		os.Exit(0)
	}

	version, _ := flag.CommandLine.GetBool("version")
	if version {
		ui.PrintVersion()
		os.Exit(0)
	}

}
