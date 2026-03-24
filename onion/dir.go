package onion

import (
	"context"
	"onionfs/core"
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

func (dn *DirNode) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {

	resolvedPath, _, err := core.ResolvePath(dn.State, dn.VirtualPath)
	if err != nil {
		return syscall.ENOENT
	}

	di, err := os.Stat(resolvedPath)
	if err != nil {
		return syscall.EIO
	}

	out.Attr.FromStat(fuse.ToStatT(di))
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
	if err != nil && !os.IsNotExist(err) {
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
		return nil, syscall.ENOENT
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
	out.EntryValid = 1

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
