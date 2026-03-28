package onion

import (
	"onionfs/core"
	"os"
	"os/signal"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

func Mount(state *core.OnionState) error {
	root := &DirNode{State: state, VirtualPath: "/"}
	server, err := fs.Mount(state.MountPoint, root, &fs.Options{
		MountOptions: fuse.MountOptions{
			Debug:  state.Debug,
			FsName: "onionfs",
			Name:   "onionfs",
		},
	})
	if err != nil {
		return err
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		server.Unmount()
	}()

	server.Wait()
	return nil
}
