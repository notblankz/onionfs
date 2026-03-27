package core

import (
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
		return "", LocationNone, syscall.ENOENT
	}

	upperPath := filepath.Join(state.UpperDir, virtualPath)
	if _, err := os.Stat(upperPath); err == nil {
		return upperPath, LocationUpper, nil
	}

	lowerPath := filepath.Join(state.LowerDir, virtualPath)
	if _, err := os.Stat(lowerPath); err == nil {
		return lowerPath, LocationLower, nil
	}

	return "", LocationNone, syscall.ENOENT
}
