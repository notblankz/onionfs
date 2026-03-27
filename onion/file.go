package onion

import (
	"context"
	"io"
	"onionfs/core"
	"os"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type FileNode struct {
	fs.Inode
	State       *core.OnionState
	VirtualPath string
	UpperPath   string
}

var _ = (fs.NodeGetattrer)((*FileNode)(nil))
var _ = (fs.NodeSetattrer)((*FileNode)(nil))
var _ = (fs.NodeOpener)((*FileNode)(nil))
var _ = (fs.NodeReader)((*FileNode)(nil))
var _ = (fs.NodeWriter)((*FileNode)(nil))
var _ = (fs.NodeFsyncer)((*FileNode)(nil))

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

func (fn *FileNode) Setattr(ctx context.Context, f fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {

	// need to multiplex and handle all of these :smile:
	// SetAttrIn.Valid enum
	// const (
	// 	x FATTR_MODE         = (1 << 0)
	// 	x FATTR_UID          = (1 << 1)
	// 	x FATTR_GID          = (1 << 2)
	// 	x FATTR_SIZE         = (1 << 3)
	// 	x FATTR_ATIME        = (1 << 4)
	// 	x FATTR_MTIME        = (1 << 5)
	// 	- FATTR_FH           = (1 << 6)		(Not Supported by this function)
	// 	x FATTR_ATIME_NOW    = (1 << 7)
	// 	x FATTR_MTIME_NOW    = (1 << 8)
	// 	- FATTR_LOCKOWNER    = (1 << 9)		(Not Supported by this function)
	// 	- FATTR_CTIME        = (1 << 10)	(Not Supported by this function)
	// 	- FATTR_KILL_SUIDGID = (1 << 11)	(Not Supported by this function)
	// )

	var resolvedPath string
	var err error
	if fn.UpperPath != "" {
		resolvedPath = fn.UpperPath
	} else {
		var err error
		resolvedPath, err = core.ResolveAndCopyUp(fn.State, fn.VirtualPath)
		if err != nil {
			return syscall.EIO
		}
	}

	if (in.Valid & fuse.FATTR_SIZE) != 0 {
		err = os.Truncate(resolvedPath, int64(in.Size))
		if err != nil {
			return syscall.EIO
		}
	}

	if (in.Valid & fuse.FATTR_MODE) != 0 {
		err = os.Chmod(resolvedPath, os.FileMode(in.Mode))
		if err != nil {
			return syscall.EIO
		}
	}

	if (in.Valid&fuse.FATTR_GID) != 0 || (in.Valid&fuse.FATTR_UID) != 0 {
		gid := -1
		uid := -1
		if (in.Valid & fuse.FATTR_GID) != 0 {
			gid = int(in.Owner.Gid)
		}
		if (in.Valid & fuse.FATTR_UID) != 0 {
			uid = int(in.Owner.Uid)
		}
		err := os.Chown(resolvedPath, uid, gid)
		if err != nil {
			return syscall.EIO
		}
	}

	if (in.Valid&fuse.FATTR_ATIME) != 0 || (in.Valid&fuse.FATTR_MTIME) != 0 {
		atime := time.Time{}
		mtime := time.Time{}
		if (in.Valid & fuse.FATTR_ATIME) != 0 {
			atime = time.Unix(int64(in.Atime), int64(in.Atimensec))
		}
		if (in.Valid & fuse.FATTR_MTIME) != 0 {
			mtime = time.Unix(int64(in.Mtime), int64(in.Mtimensec))
		}
		err := os.Chtimes(resolvedPath, atime, mtime)
		if err != nil {
			return syscall.EIO
		}
	}

	if (in.Valid&fuse.FATTR_ATIME_NOW) != 0 || (in.Valid&fuse.FATTR_MTIME_NOW) != 0 {
		atimenow := time.Time{}
		mtimenow := time.Time{}
		if (in.Valid & fuse.FATTR_ATIME_NOW) != 0 {
			atimenow = time.Now()
		}
		if (in.Valid & fuse.FATTR_MTIME_NOW) != 0 {
			mtimenow = time.Now()
		}
		err := os.Chtimes(resolvedPath, atimenow, mtimenow)
		if err != nil {
			return syscall.EIO
		}
	}

	// set the fields in (out *fs.AttrOut)
	out.AttrValid = 1
	fi, err := os.Stat(resolvedPath)
	if err != nil {
		return syscall.EIO
	}
	fiStatT := fuse.ToStatT(fi)
	out.Attr.FromStat(fiStatT)

	return 0
}

func (fn *FileNode) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {

	var resolvedPath string
	var err error

	// mask all bits except the WR and RDWR bits to check whether its a write request
	isWrite := (flags&syscall.O_WRONLY) != 0 || (flags&syscall.O_RDWR) != 0 || (flags&syscall.O_APPEND) != 0

	if isWrite {
		resolvedPath, err = core.ResolveAndCopyUp(fn.State, fn.VirtualPath)
	} else {
		resolvedPath, _, err = core.ResolvePath(fn.State, fn.VirtualPath)
	}

	f, err := os.OpenFile(resolvedPath, int(flags), 0)
	if err != nil {
		return nil, 0, syscall.EIO
	}

	fn.UpperPath = resolvedPath

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

	// Write uses Pwrite instead of WriteAt since WriteAt does not allow fd with O_APPEND flag set
	// Pwrite ignores the O_APPEND flag on the fd
	// as the kernel calculates and gives us the correct offset to WriteAt the behaviour of write doesn't change

	if f != nil {
		// typecast fs.FileHandle to os.File to access WriteAt
		file := f.(*os.File)
		n, err := syscall.Pwrite(int(file.Fd()), data, off)
		if err != nil {
			return 0, syscall.EIO
		}
		return uint32(n), 0
	}

	// No open fd given so we open our own to write into it
	var resolvedPath string
	if fn.UpperPath != "" {
		resolvedPath = fn.UpperPath
	} else {
		var err error
		resolvedPath, err = core.ResolveAndCopyUp(fn.State, fn.VirtualPath)
		if err != nil {
			return 0, syscall.EIO
		}
	}

	file, err := os.OpenFile(resolvedPath, os.O_WRONLY, 0)
	if err != nil {
		return 0, syscall.EIO
	}
	defer file.Close()

	n, err := syscall.Pwrite(int(file.Fd()), data, off)
	if err != nil {
		return 0, syscall.EIO
	}

	return uint32(n), 0
}

func (fn *FileNode) Fsync(ctx context.Context, f fs.FileHandle, flags uint32) syscall.Errno {
	if f != nil {
		file := f.(*os.File)
		err := syscall.Fsync(int(file.Fd()))
		if err != nil {
			return syscall.EIO
		}
		return 0
	}

	resolvedPath, err := core.ResolveAndCopyUp(fn.State, fn.VirtualPath)
	if err != nil {
		return syscall.EIO
	}

	file, err := os.OpenFile(resolvedPath, os.O_WRONLY, 0)
	if err != nil {
		return syscall.EIO
	}
	defer file.Close()

	err = syscall.Fsync(int(file.Fd()))
	if err != nil {
		return syscall.EIO
	}

	return 0
}
