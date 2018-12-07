package main

import (
	"context"
	"log"
	"path"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type IPFSRootNode struct {
	nodefs.Node
}

func (n *IPFSRootNode) Lookup(out *fuse.Attr, name string, ctx *fuse.Context) (*nodefs.Inode, fuse.Status) {
	return lookupIPFS(n.Inode(), out, name, ctx)
}

func lookupIPFS(inode *nodefs.Inode, out *fuse.Attr, name string, ctx *fuse.Context) (*nodefs.Inode, fuse.Status) {
	stat, err := Stat(context.TODO(), "/ipfs/"+name)
	if err != nil {
		log.Println("Lookup", "/ipfs/"+name, err)
		return nil, fuse.EIO
	}
	if stat == nil {
		inode.RmChild(path.Base(name))
		return nil, fuse.ENOENT
	}

	var entries *UnixFSList
	out.Size = stat.Size
	out.Blocks = out.Size
	out.Blksize = 1
	out.Mode = 0444
	if stat.Type == "directory" {
		out.Mode |= 0111 | fuse.S_IFDIR
		entries, err = ListImmutable(context.TODO(), "/ipfs/"+name+"/")
		if err != nil {
			log.Println("Lookup", "/ipfs/"+name, err)
			return nil, fuse.EIO
		}
		if entries == nil {
			inode.RmChild(path.Base(name))
			return nil, fuse.ENOENT
		}
		out.Size = uint64(len(entries.Entries))
		out.Blocks = out.Size
	} else if stat.Type == "file" {
		out.Mode |= fuse.S_IFREG
	}

	return inode.NewChild(name, out.IsDir(), &IPFSNode{
		Node:    nodefs.NewDefaultNode(),
		Hash:    stat.Hash,
		Stat:    stat,
		Entries: entries,
	}), fuse.OK
}

func (n *IPFSRootNode) OpenDir(ctx *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	return nil, fuse.EPERM
}

func (n *IPFSRootNode) GetXAttr(attribute string, ctx *fuse.Context) (data []byte, code fuse.Status) {
	return nil, fuse.ENOATTR
}

func (n *IPFSRootNode) ListXAttr(ctx *fuse.Context) (attrs []string, code fuse.Status) {
	return nil, fuse.OK
}

func (n *IPFSRootNode) GetAttr(out *fuse.Attr, file nodefs.File, ctx *fuse.Context) fuse.Status {
	out.Mode = 0111 | fuse.S_IFDIR
	return fuse.OK
}
