package onion

import (
	"context"
	"errors"
	"onionfs/core"
	"onionfs/ui"
	"os"
	"path/filepath"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type DirNode struct {
	fs.Inode
	State       *core.OnionState
	VirtualPath string
}

var _ = (fs.NodeGetattrer)((*DirNode)(nil))
var _ = (fs.NodeReaddirer)((*DirNode)(nil))
var _ = (fs.NodeLookuper)((*DirNode)(nil))
var _ = (fs.NodeMkdirer)((*DirNode)(nil))
var _ = (fs.NodeCreater)((*DirNode)(nil))
var _ = (fs.NodeUnlinker)((*DirNode)(nil))
var _ = (fs.NodeRmdirer)((*DirNode)(nil))
var _ = (fs.NodeRenamer)((*DirNode)(nil))

func (dn *DirNode) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {

	resolvedPath, _, err := core.ResolvePath(dn.State, dn.VirtualPath)
	if err != nil {
		return syscall.ENOENT
	}

	di, err := os.Stat(resolvedPath)
	if err != nil {
		return syscall.EIO
	}

	out.Attr = *fuse.ToAttr(di)
	out.AttrValid = 1

	return 0
}

func (dn *DirNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {

	// read all entries from upper dir
	// exclude the .wh. name (include if --show-meta flag is enabled) -> add into whiteout set
	// read all entries from the lower dir
	// when reading entries from the lower dir we need to take care of these things
	// - if the entry exists in the upper dir (i.e. already existing in entried -> skip)
	// - skip if it exists in the whiteout set
	// - skip if the names start with .wh.
	// merge and convert to fs.DirStream
	seen := make(map[string]bool)
	whiteout := make(map[string]bool)
	var entries []fuse.DirEntry

	// upper dir path
	upperDir := filepath.Join(dn.State.UpperDir, dn.VirtualPath)
	// all entries in the upper directory
	upperDirEntries, err := os.ReadDir(upperDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, syscall.ENOENT
	}
	for _, entry := range upperDirEntries {
		switch core.IsWhiteoutFile(entry.Name()) {
		case true:
			if !dn.State.HideMeta {
				fuseDirEntry, err := toFuseDirEntry(entry)
				if err != nil {
					return nil, syscall.EIO
				}
				entries = append(entries, fuseDirEntry)
				seen[core.WhiteoutTarget(entry.Name())] = true
				seen[entry.Name()] = true
			}
			// whiteout file skip & add to whiteout
			whiteout[core.WhiteoutTarget(entry.Name())] = true
		case false:
			// convert os.DirEntry to fuse.DirEntry
			fuseDirEntry, err := toFuseDirEntry(entry)
			if err != nil {
				return nil, syscall.EIO
			}
			// add to entries
			entries = append(entries, fuseDirEntry)
			// add to seen
			seen[entry.Name()] = true
		}
	}

	// lower dir path
	lowerDir := filepath.Join(dn.State.LowerDir, dn.VirtualPath)
	lowerDirEntries, err := os.ReadDir(lowerDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, syscall.ENOENT
	}
	for _, entry := range lowerDirEntries {
		// check each entry
		// - if it exists in entries skip it
		// - if it exists in whiteout set skip it
		// - else add it to the entries
		if seen[entry.Name()] {
			continue
		} else if whiteout[entry.Name()] {
			continue
		} else {
			fuseDirEntry, err := toFuseDirEntry(entry)
			if err != nil {
				return nil, syscall.EIO
			}
			entries = append(entries, fuseDirEntry)
		}
	}

	return fs.NewListDirStream(entries), 0
}

func (dn *DirNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {

	// create the full childVirtualPath i.e. /dn.VirtualPath/name
	// then resolve the childVirtualPath so that we get the correct path which can be either core.LocationUpper or core.LocationLower
	// fill out the fuse.EntryOut struct with infromation from the Stat_t struct of the child
	// check if child is a file or dir
	// - if its a file then create a new FileNode (with the necessary info) -> create a NewInode from the caller node (dn *DirNode)
	// (The lookup call is done on the parent node, hence if the lookup suceeds the parent
	// node must create a NewInode with the child's INODE number which will be added to the table)
	// For Eg: to see /mnt/etc/hosts -> kernel does mnt.Lookup(etc) -> etc.Lookup(hosts) -> which then creates a new Entry for hosts new INODE
	// - if its a dir then do the same thing with a DirNode

	if core.IsWhiteoutFile(name) {
		if dn.State.HideMeta {
			return nil, syscall.ENOENT
		}

		// we need to return the .wh. file INODE if the HideMeta flag is set to false
		whiteoutRealPath := filepath.Join(dn.State.UpperDir, dn.VirtualPath, name)
		fi, err := os.Stat(whiteoutRealPath)
		if err != nil {
			return nil, syscall.ENOENT
		}
		stat := fuse.ToStatT(fi)
		out.Attr.FromStat(stat)
		out.AttrValid = 1
		out.EntryValid = 0
		child := &FileNode{
			State:       dn.State,
			VirtualPath: filepath.Join(dn.VirtualPath, name),
		}
		return dn.NewInode(ctx, child, fs.StableAttr{
			Mode: fuse.S_IFREG,
			Ino:  stat.Ino,
		}), 0
	}

	childVirtualPath := filepath.Join(dn.VirtualPath, name)

	childResolvedPath, _, err := core.ResolvePath(dn.State, childVirtualPath)
	if err != nil {
		return nil, syscall.ENOENT
	}

	fi, err := os.Stat(childResolvedPath)
	if err != nil {
		return nil, syscall.ENOENT
	}

	stat := fuse.ToStatT(fi)
	out.Attr.FromStat(stat)
	out.AttrValid = 1

	if fi.IsDir() {
		child := &DirNode{
			State:       dn.State,
			VirtualPath: childVirtualPath,
		}
		stableAttr := fs.StableAttr{
			Mode: fuse.S_IFDIR,
			Ino:  stat.Ino,
		}
		return dn.NewInode(ctx, child, stableAttr), 0
	} else {
		child := &FileNode{
			State:       dn.State,
			VirtualPath: childVirtualPath,
		}
		stableAttr := fs.StableAttr{
			Mode: fuse.S_IFREG,
			Ino:  stat.Ino,
		}
		return dn.NewInode(ctx, child, stableAttr), 0
	}
}

func (dn *DirNode) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {

	newDirVirtualPath := filepath.Join(dn.VirtualPath, name)

	fullPath := filepath.Join(dn.State.UpperDir, newDirVirtualPath)

	// check if the dir being created already exists in lower dir/upper dir
	_, _, resolvedErr := core.ResolvePath(dn.State, newDirVirtualPath)
	if resolvedErr == nil {
		return nil, syscall.EEXIST
	}

	// check for a whiteout marker for the directory and remove it before proceeding to create dir
	whiteoutFullPath := filepath.Join(dn.State.UpperDir, dn.VirtualPath, core.WhiteoutName(name))
	removeErr := os.Remove(whiteoutFullPath)
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return nil, syscall.EIO
	}

	err := os.Mkdir(fullPath, os.FileMode(mode))
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, syscall.EEXIST
		}
		return nil, syscall.EIO
	}

	out.AttrValid = 1
	out.EntryValid = 1

	di, err := os.Stat(fullPath)
	if err != nil {
		return nil, syscall.EIO
	}
	diStatT := fuse.ToStatT(di)
	out.Attr.FromStat(diStatT)

	newDir := &DirNode{
		State:       dn.State,
		VirtualPath: newDirVirtualPath,
	}
	stableAttr := fs.StableAttr{
		Mode: fuse.S_IFDIR,
		Ino:  diStatT.Ino,
	}

	return dn.NewInode(ctx, newDir, stableAttr), 0
}

func (dn *DirNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (node *fs.Inode, fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {

	newFileVirtualPath := filepath.Join(dn.VirtualPath, name)

	fullPath := filepath.Join(dn.State.UpperDir, newFileVirtualPath)

	// same check as Mkdir - checking for existence in upper dir/lower dir
	_, _, resolvedPath := core.ResolvePath(dn.State, newFileVirtualPath)
	if resolvedPath == nil {
		return nil, nil, 0, syscall.EEXIST
	}

	// here we remove the whitout markers
	whiteoutFullPath := filepath.Join(dn.State.UpperDir, dn.VirtualPath, core.WhiteoutName(name))
	removeErr := os.Remove(whiteoutFullPath)
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return nil, nil, 0, syscall.EIO
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, nil, 0, syscall.EIO
	}

	file, err := os.OpenFile(fullPath, int(flags)|os.O_CREATE, os.FileMode(mode))
	if err != nil {
		return nil, nil, 0, syscall.EIO
	}

	out.AttrValid = 1
	out.EntryValid = 1

	fi, err := os.Stat(fullPath)
	if err != nil {
		return nil, nil, 0, syscall.EIO
	}
	fiStatT := fuse.ToStatT(fi)
	out.Attr.FromStat(fiStatT)

	newFile := &FileNode{
		State:       dn.State,
		VirtualPath: newFileVirtualPath,
	}
	stableAttr := fs.StableAttr{
		Mode: fuse.S_IFREG,
		Ino:  fiStatT.Ino,
	}

	return dn.NewInode(ctx, newFile, stableAttr), file, fuse.FOPEN_KEEP_CACHE, 0
}

func (dn *DirNode) Unlink(ctx context.Context, name string) syscall.Errno {

	toDeleteVirtualPath := filepath.Join(dn.VirtualPath, name)

	ui.Info("[UNLINK] Received File to Delete: %s", toDeleteVirtualPath)

	err := core.CreateWhiteout(dn.State, toDeleteVirtualPath)
	if err != 0 {
		return err
	}

	ui.Info("[UNLINK] Successfully deleted File %s", toDeleteVirtualPath)

	return 0
}

func (dn *DirNode) Rmdir(ctx context.Context, name string) syscall.Errno {

	toDeleteVirtualPath := filepath.Join(dn.VirtualPath, name)

	ui.Info("[RMDIR] Received Directory to Delete: %s", toDeleteVirtualPath)

	isEmpty, err := dn.isDirEmpty(ctx, name)
	if err != 0 {
		return err
	}
	if !isEmpty {
		ui.Error("[RMDIR] %s is not empty", toDeleteVirtualPath)
		return syscall.ENOTEMPTY
	}

	err = core.CreateWhiteout(dn.State, toDeleteVirtualPath)
	if err != 0 {
		return err
	}

	ui.Info("[RMDIR] Successfully removed directory: %s", toDeleteVirtualPath)

	return 0
}

func (dn *DirNode) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {

	newParentNode, ok := newParent.(*DirNode)
	if !ok {
		return syscall.EIO
	}

	srcVirtualPath := filepath.Join(dn.VirtualPath, name)
	dstVirtualPath := filepath.Join(newParentNode.VirtualPath, newName)

	ui.Info("[RENAME] %s to %s", srcVirtualPath, dstVirtualPath)

	srcResolvedPath, err := core.ResolveAndCopyUp(dn.State, srcVirtualPath)
	if err != nil {
		return syscall.ENOENT
	}

	if err := os.Rename(srcResolvedPath, filepath.Join(dn.State.UpperDir, dstVirtualPath)); err != nil {
		return syscall.EIO
	}

	ui.Info("[RENAME] Successfully renamed %s to %s", srcVirtualPath, dstVirtualPath)

	return 0
}

// -- HELPER --
func toFuseDirEntry(entry os.DirEntry) (fuse.DirEntry, error) {
	entryInfo, err := entry.Info()
	if err != nil {
		return fuse.DirEntry{}, err
	}
	entryInfoStatT := fuse.ToStatT(entryInfo)
	fuseDirEntry := fuse.DirEntry{
		Mode: uint32(entryInfo.Mode()),
		Name: entry.Name(),
		Ino:  entryInfoStatT.Ino,
		// Offset is set by fuse directly
	}

	return fuseDirEntry, nil
}

func (dn *DirNode) isDirEmpty(ctx context.Context, name string) (bool, syscall.Errno) {

	// create a dummy child node with the children in its virtual path
	child := &DirNode{
		State:       dn.State,
		VirtualPath: filepath.Join(dn.VirtualPath, name),
	}

	// use the OnionFS implemented Read dir (instead of os.ReadDir)
	// - this makes sure we read the merged dir i.e. combining R/W and R/O layers + skipping whiteout files
	dirStream, err := child.Readdir(ctx)
	if err != 0 {
		return false, err
	}

	// traverse the child directory to check if its empty
	for dirStream.HasNext() {
		entry, err := dirStream.Next()
		if err != 0 {
			return false, err
		}
		// if we encounter anything except "." <- This directory or Parent directory -> ".." entries
		// - that means the child directory has children hence return false
		if entry.Name != "." && entry.Name != ".." {
			return false, 0
		}
	}

	return true, 0
}
