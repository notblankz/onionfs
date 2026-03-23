package core

import (
	"os"
	"path/filepath"
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

func CreateWhiteout(state *OnionState, virtualPath string) error {
	// this will be called only for deletion in lowerdir
	// since we can directly unlink if it is in upper
	whiteoutDir := filepath.Join(state.UpperDir, filepath.Dir(virtualPath))
	if err := os.MkdirAll(whiteoutDir, 0755); err != nil {
		return err
	}

	// get whiteout file absolute path
	filename := filepath.Base(virtualPath)
	whiteoutFile := filepath.Join(whiteoutDir, WhiteoutName(filename))
	// create the whiteout file - marking the file in lower dir deleted for the merged fs
	f, err := os.Create(whiteoutFile)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}
