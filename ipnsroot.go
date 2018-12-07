package main

import (
	"context"
	"log"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type IPNSRootNode struct {
	nodefs.Node
}

func (n *IPNSRootNode) Lookup(out *fuse.Attr, name string, ctx *fuse.Context) (*nodefs.Inode, fuse.Status) {
	dest, err := Resolve(context.TODO(), name)
	if err != nil {
		log.Println("Lookup", "/ipns/"+name, err)
		return nil, fuse.EIO
	}
	if dest == "" {
		return nil, fuse.ENOENT
	}

	out.Mode = 0444 | fuse.S_IFLNK
	return n.Inode().NewChild(name, false, &IPNSNode{
		Node: nodefs.NewDefaultNode(),
		Dest: dest,
	}), fuse.OK
}
func (n *IPNSRootNode) OpenDir(ctx *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	return nil, fuse.EPERM
}
func (n *IPNSRootNode) GetAttr(out *fuse.Attr, file nodefs.File, ctx *fuse.Context) fuse.Status {
	out.Mode = 0111 | fuse.S_IFDIR
	return fuse.OK
}
