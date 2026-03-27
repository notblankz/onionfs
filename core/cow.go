package core

import (
	"io"
	"onionfs/ui"
	"os"
	"path/filepath"
)

func CopyUp(state *OnionState, virtualPath string) (string, error) {

	lowerDirFilepath := filepath.Join(state.LowerDir, virtualPath)
	upperDirFilepath := filepath.Join(state.UpperDir, virtualPath)

	// stat the file in the lower dir to get the permission bits
	fi, err := os.Stat(lowerDirFilepath)
	if err != nil {
		return "", err
	}
	filePermBits := fi.Mode().Perm()

	// stat the dir to get the permission bits of the dir
	dirFi, err := os.Stat(filepath.Dir(lowerDirFilepath))
	if err != nil {
		return "", err
	}
	dirPermBits := dirFi.Mode().Perm()

	// create the dir need to make sure to preserve the perm
	err = os.MkdirAll(filepath.Dir(upperDirFilepath), dirPermBits)
	if err != nil {
		return "", err
	}

	// open source file (in lower dir)
	sourceFile, err := os.Open(lowerDirFilepath)
	if err != nil {
		return "", err
	}
	defer sourceFile.Close()

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
	err = os.Chmod(upperDirFilepath, filePermBits)
	if err != nil {
		return "", err
	}

	ui.Info("[CORE][CopyUp]", "%s copied up to upper directory", virtualPath)

	return upperDirFilepath, nil
}

func ResolveAndCopyUp(state *OnionState, virtualPath string) (string, error) {

	resolvedPath, location, err := ResolvePath(state, virtualPath)
	if err != nil {
		return "", err
	}

	if location == LocationLower {
		resolvedPath, err = CopyUp(state, virtualPath)
		if err != nil {
			return "", err
		}
	}

	return resolvedPath, nil
}
