package onion

import (
	"onionfs/core"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

func Mount(state *core.OnionState) error {
	root := &DirNode{State: state, VirtualPath: "/"}
	server, err := fs.Mount(state.MountPoint, root, &fs.Options{
		MountOptions: fuse.MountOptions{
			Debug:  false,
			FsName: "onionfs",
			Name:   "onionfs",
		},
	})
	if err != nil {
		return err
	}
	server.Wait()
	return nil
}
