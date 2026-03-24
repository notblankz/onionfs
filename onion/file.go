package onion

import (
	"context"
	"io"
	"onionfs/core"
	"os"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type FileNode struct {
	fs.Inode
	State       *core.OnionState
	VirtualPath string
}

var _ = (fs.NodeGetattrer)((*FileNode)(nil))
var _ = (fs.NodeOpener)((*FileNode)(nil))
var _ = (fs.NodeReader)((*FileNode)(nil))
var _ = (fs.NodeWriter)((*FileNode)(nil))

// function to get Metadata of INODE and pass it to the kernel
func (fn *FileNode) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {

	resolvedPath, _, err := core.ResolvePath(fn.State, fn.VirtualPath)
	if err != nil {
		return syscall.ENOENT
	}

	fileStatInfo, err := os.Stat(resolvedPath)
	if err != nil {
		return syscall.ENOENT
	}

	out.Attr.FromStat(fuse.ToStatT(fileStatInfo))
	out.AttrValid = 1

	return 0
}

func (fn *FileNode) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {

	// resolve path
	resolvedPath, location, err := core.ResolvePath(fn.State, fn.VirtualPath)
	if err != nil {
		return nil, 0, syscall.ENOENT
	}

	// mask all bits except the WR and RDWR bits to check whether its a write request
	isWrite := (flags&syscall.O_WRONLY) != 0 || (flags&syscall.O_RDWR) != 0

	if isWrite && location == core.LocationLower {
		resolvedPath, err = core.CopyUp(fn.State, fn.VirtualPath)
		if err != nil {
			return nil, 0, syscall.EIO
		}
	}

	f, err := os.OpenFile(resolvedPath, int(flags), 0)
	if err != nil {
		return nil, 0, syscall.EIO
	}
	return f, fuse.FOPEN_KEEP_CACHE, 0
}

func (fn *FileNode) Read(ctx context.Context, f fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	// typecast fs.FileHandle to os.File to access ReadAt
	file := f.(*os.File)

	n, err := file.ReadAt(dest, off)
	if err != nil && err != io.EOF {
		return nil, syscall.EIO
	}

	return fuse.ReadResultData(dest[:n]), 0
}

func (fn *FileNode) Write(ctx context.Context, f fs.FileHandle, data []byte, off int64) (written uint32, errno syscall.Errno) {
	// typecast fs.FileHandle to os.File to access WriteAt
	file := f.(*os.File)

	n, err := file.WriteAt(data, off)
	if err != nil {
		return 0, syscall.EIO
	}

	return uint32(n), 0
}
