package main

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type IPFSNode struct {
	nodefs.Node
	Hash    string
	Stat    *UnixFSStat
	Entries *UnixFSList
}

func (n *IPFSNode) Lookup(out *fuse.Attr, name string, ctx *fuse.Context) (*nodefs.Inode, fuse.Status) {
	return lookupIPFS(n.Inode(), out, n.Hash+"/"+name, ctx)
}

func (n *IPFSNode) Open(flags uint32, ctx *fuse.Context) (nodefs.File, fuse.Status) {
	if n.Entries != nil {
		return nil, fuse.EISDIR
	}

	return &ReadOnlyFile{File: nodefs.NewDefaultFile(), Hash: n.Hash}, fuse.OK
}

func (n *IPFSNode) OpenDir(ctx *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	if n.Entries == nil {
		return nil, fuse.ENOTDIR
	}

	entries := make([]fuse.DirEntry, len(n.Entries.Entries))
	for i, e := range n.Entries.Entries {
		var mode uint32
		if e.Type == Directory {
			mode = fuse.S_IFDIR
		} else {
			mode = fuse.S_IFREG
		}
		entries[i] = fuse.DirEntry{
			Name: e.Name,
			Mode: mode,
		}
	}

	return entries, fuse.OK
}

func (n *IPFSNode) GetAttr(out *fuse.Attr, file nodefs.File, ctx *fuse.Context) fuse.Status {
	out.Mtime = 1
	out.Ctime = 1

	out.Mode = 0444
	if n.Entries != nil {
		out.Mode |= 0111 | fuse.S_IFDIR
		out.Size = uint64(len(n.Entries.Entries))
	} else {
		out.Mode |= fuse.S_IFREG
		out.Size = n.Stat.Size
	}

	out.Blocks = out.Size
	out.Blksize = 1

	return fuse.OK
}
