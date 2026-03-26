package core

import (
	"errors"
	"onionfs/ui"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func WhiteoutName(filename string) string {
	return ".wh." + filename
}

func IsWhitedOut(state *OnionState, virtualPath string) bool {
	// .wh.<filename>
	whiteoutPath := filepath.Join(state.UpperDir, filepath.Dir(virtualPath), ".wh."+filepath.Base(virtualPath))
	if _, err := os.Stat(whiteoutPath); err == nil {
		return true
	}

	return false
}

func IsWhiteoutFile(name string) bool {
	return strings.HasPrefix(name, ".wh.")
}

func WhiteoutTarget(name string) string {
	processedName, ok := strings.CutPrefix(name, ".wh.")
	if !ok {
		return name
	}

	return processedName
}

func CreateWhiteout(state *OnionState, virtualPath string) syscall.Errno {
	// check the existence in upper directory - if it does then just delete it
	// now check its existence in lower dir - if it does then make .wh. file
	// and the checks should not err if file not found

	upperDirPath := filepath.Join(state.UpperDir, virtualPath)
	lowerDirPath := filepath.Join(state.LowerDir, virtualPath)

	ui.Info("Upper Dir Path to Delete: %s", upperDirPath)
	ui.Info("Lower Dir Path to Delete: %s", lowerDirPath)

	_, errUpper := os.Stat(upperDirPath)
	upperExists := errUpper == nil
	if errUpper != nil && !errors.Is(errUpper, os.ErrNotExist) {
		return syscall.EIO
	}

	_, errLower := os.Stat(lowerDirPath)
	lowerExists := errLower == nil
	if errLower != nil && !errors.Is(errLower, os.ErrNotExist) {
		return syscall.EIO
	}

	if !upperExists && !lowerExists {
		return syscall.ENOENT
	}

	if upperExists {
		err := os.Remove(upperDirPath)
		if err != nil {
			return syscall.EIO
		}
		ui.Info("Finished Deleting: %s", upperDirPath)
	}

	if lowerExists {
		whiteoutDir := filepath.Join(state.UpperDir, filepath.Dir(virtualPath))
		if err := os.MkdirAll(whiteoutDir, 0755); err != nil {
			return syscall.EIO
		}

		whiteoutPath := filepath.Join(whiteoutDir, WhiteoutName(filepath.Base(virtualPath)))

		f, err := os.Create(whiteoutPath)
		if err != nil {
			return syscall.EIO
		}
		f.Close()
		ui.Info("Finished creating whiteout file: %s", whiteoutPath)
	}

	ui.Info("Successfully deleted: %s", virtualPath)

	return 0
}
