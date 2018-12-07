package main

import (
	"context"
	"log"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type UnixFSFile struct {
	nodefs.File
	Node *UnixFSNode
}

func (f *UnixFSFile) Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status) {
	resp, err := ipfs.Request("files/read", f.Node.Path).Option("offset", off).Option("count", len(dest)).Option("flush", false).Send(context.TODO())
	if err == nil && resp.Error != nil {
		err = resp.Error
	}
	if err != nil {
		log.Println("Read", f.Node.Path, err)
		return nil, fuse.EIO
	}

	n, err := readFull(resp.Output, dest)
	if e := resp.Close(); err == nil {
		err = e
	}
	result := fuse.ReadResultData(dest[:n])
	if err != nil {
		log.Println("Read", f.Node.Path, err)
		return result, fuse.EIO
	}

	return result, fuse.OK
}

func (f *UnixFSFile) Write(data []byte, off int64) (uint32, fuse.Status) {
	resp, err := attachFile(ipfs.Request("files/write", f.Node.Path), data).Option("flush", false).Option("offset", off).Option("raw-leaves", true).Send(context.TODO())
	if err == nil {
		err = resp.Close()
	}
	if err == nil && resp.Error != nil {
		err = resp.Error
	}

	if err != nil {
		log.Println("Write", f.Node.Path)
		return 0, fuse.EIO
	}

	return uint32(len(data)), fuse.OK
}

func (f *UnixFSFile) Flush() fuse.Status {
	resp, err := ipfs.Request("files/flush", f.Node.Path).Send(context.TODO())
	if err == nil {
		err = resp.Close()
	}
	if err == nil && resp.Error != nil {
		err = resp.Error
	}

	if err != nil {
		log.Println("Flush", f.Node.Path, err)
		return fuse.EIO
	}
	return fuse.OK
}

func (f *UnixFSFile) Fsync(flags int) fuse.Status {
	return f.Flush()
}

func (f *UnixFSFile) Truncate(size uint64) fuse.Status {
	if size == 0 {
		resp, err := attachFile(ipfs.Request("files/write", f.Node.Path), nil).Option("flush", false).Option("truncate", true).Option("raw-leaves", true).Send(context.TODO())
		if err == nil {
			err = resp.Close()
		}
		if err == nil && resp.Error != nil {
			err = resp.Error
		}

		if err != nil {
			log.Println("Truncate", f.Node.Path)
			return fuse.EIO
		}

		return fuse.OK
	}

	return fuse.ENOSYS
}

func (f *UnixFSFile) GetAttr(out *fuse.Attr) fuse.Status {
	return f.Node.GetAttr(out, f, &fuse.Context{})
}
