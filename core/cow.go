package core

import (
	"io"
	"os"
	"path/filepath"
)

func CopyUp(state *OnionState, virtualPath string) (string, error) {

	lowerDirFilepath := filepath.Join(state.LowerDir, virtualPath)
	fi, err := os.Stat(lowerDirFilepath)
	if err != nil {
		return "", err
	}
	permBits := fi.Mode().Perm()

	// create the dir need to make sure to preserve the perms
	err = os.MkdirAll(filepath.Dir(lowerDirFilepath), permBits)
	if err != nil {
		return "", err
	}

	// open source file (in lower dir)
	sourceFile, err := os.Open(lowerDirFilepath)
	if err != nil {
		return "", err
	}
	defer sourceFile.Close()

	// copy the actual file to upper dir
	upperDirFilepath := filepath.Join(state.UpperDir, virtualPath)
	upperDirFile, err := os.Create(upperDirFilepath)
	if err != nil {
		return "", err
	}
	defer upperDirFile.Close()

	// copy contents of the lower dir file to the upper R/W layer
	_, err = io.Copy(upperDirFile, sourceFile)
	if err != nil {
		return "", err
	}

	// change permission bits to reflect the original content
	err = os.Chmod(upperDirFilepath, permBits)
	if err != nil {
		return "", err
	}

	return upperDirFilepath, nil
}
