package main

import (
	"fmt"
	"onionfs/core"
	"onionfs/onion"
	"onionfs/ui"
	"os"
	"path/filepath"

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

	version, _ := flag.CommandLine.GetBool("version")
	if version {
		ui.PrintVersion()
	}

	lowerDir, _ := flag.CommandLine.GetString("lower")
	upperDir, _ := flag.CommandLine.GetString("upper")
	mountpoint, _ := flag.CommandLine.GetString("mountpoint")

	if len(lowerDir) == 0 || len(upperDir) == 0 || len(mountpoint) == 0 {
		ui.Fatal("Missing required parameters")
		ui.PrintHelp()
	}

	lowerDirAbs, err := validatePath(lowerDir)
	if err != nil {
		ui.Fatal("%v", err)
		os.Exit(0)
	}

	upperDirAbs, err := validatePath(upperDir)
	if err != nil {
		ui.Fatal("%v", err)
		os.Exit(0)
	}

	mountpointAbs, err := validatePath(mountpoint)
	if err != nil {
		ui.Fatal("%v", err)
	}

	// Init an OnionState and populate with values
	onionstate := &core.OnionState{
		LowerDir:   lowerDirAbs,
		UpperDir:   upperDirAbs,
		MountPoint: mountpointAbs,
		CoW:        !noCow,
		HideMeta:   !showMeta,
		Foreground: !daemon,
	}

	onion.Mount(onionstate)
}

func validatePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %v", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", fmt.Errorf("path does not exist: %s", absPath)
	}
	return absPath, nil
}
