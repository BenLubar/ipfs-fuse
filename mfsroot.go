package main

import (
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type UnixFSRootNode struct {
	UnixFSNode
}

func (n *UnixFSRootNode) OnMount(conn *nodefs.FileSystemConnector) {
	n.Inode().NewChild("ipfs", true, ipfsRoot)
	n.Inode().NewChild("ipns", true, ipnsRoot)
}
