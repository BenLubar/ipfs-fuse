package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

var flagMountPoint = flag.String("mount", filepath.Join(os.Getenv("HOME"), "ipfs-fuse"), "mount point")

var ufsRoot *UnixFSRootNode
var ipfsRoot *IPFSRootNode
var ipnsRoot *IPNSRootNode

func main() {
	flag.Parse()

	ufsRoot = &UnixFSRootNode{UnixFSNode: UnixFSNode{Node: nodefs.NewDefaultNode(), Path: "/"}}
	ipfsRoot = &IPFSRootNode{Node: nodefs.NewDefaultNode()}
	ipnsRoot = &IPNSRootNode{Node: nodefs.NewDefaultNode()}

	opts := nodefs.NewOptions()
	conn := nodefs.NewFileSystemConnector(ufsRoot, opts)
	server, err := fuse.NewServer(conn.RawFS(), *flagMountPoint, &fuse.MountOptions{
		AllowOther:           true,
		FsName:               "ipfs",
		IgnoreSecurityLabels: true,
	})
	if err != nil {
		panic(err)
	}

	server.Serve()
}
