package main

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type IPNSNode struct {
	nodefs.Node
	Dest string
}

func (n *IPNSNode) Readlink(ctx *fuse.Context) ([]byte, fuse.Status) {
	return []byte(".." + n.Dest), fuse.OK
}

func (n *IPNSNode) GetAttr(out *fuse.Attr, file nodefs.File, ctx *fuse.Context) fuse.Status {
	out.Mode = fuse.S_IFLNK | 0444
	return fuse.OK
}
