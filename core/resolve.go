package core

import (
	"onionfs/ui"
	"os"
	"path/filepath"
	"syscall"
)

type Location int

const (
	LocationUpper Location = iota
	LocationLower
	LocationNone
)

func ResolvePath(state *OnionState, virtualPath string) (string, Location, error) {

	// check for whiteout files
	// check upper dir for existence of file
	// check lower dir for existence of file
	// otherwise return ENOENT

	if ok := IsWhitedOut(state, virtualPath); ok {
		ui.Info("[CORE][ResolvePath]", "%s is whited out", virtualPath)
		return "", LocationNone, syscall.ENOENT
	}

	upperPath := filepath.Join(state.UpperDir, virtualPath)
	if _, err := os.Stat(upperPath); err == nil {
		ui.Info("[CORE][ResolvePath]", "found %s in upper directory", virtualPath)
		return upperPath, LocationUpper, nil
	}

	lowerPath := filepath.Join(state.LowerDir, virtualPath)
	if _, err := os.Stat(lowerPath); err == nil {
		ui.Info("[CORE][ResolvePath]", "found %s in lower directory", virtualPath)
		return lowerPath, LocationLower, nil
	}

	ui.Info("[CORE][ResolvePath]", "could not find %s returning ENOENT", virtualPath)

	return "", LocationNone, syscall.ENOENT
}
