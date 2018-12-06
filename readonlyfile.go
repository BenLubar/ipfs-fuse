package main

import (
	"context"
	"io"
	"log"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type ReadOnlyFile struct {
	nodefs.File
	Hash string
}

// readFull is like io.ReadFull, but never considers EOF to be an error.
func readFull(r io.Reader, b []byte) (int, error) {
	n, err := io.ReadFull(r, b)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		err = nil
	}
	return n, err
}

func (f *ReadOnlyFile) Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status) {
	resp, err := ipfs.Request("cat", "/ipfs/"+f.Hash).Option("offset", off).Option("length", len(dest)).Send(context.TODO())
	if err == nil && resp.Error != nil {
		err = resp.Error
	}
	if err != nil {
		log.Println("Read", "/ipfs/"+f.Hash, err)
		return nil, fuse.EIO
	}

	n, err := readFull(resp.Output, dest)
	result := fuse.ReadResultData(dest[:n])

	if e := resp.Close(); err != nil || e != nil {
		if err == nil {
			err = e
		}
		log.Println("Read", "/ipfs/"+f.Hash, err)
		return result, fuse.EIO
	}

	return result, fuse.OK
}

func (f *ReadOnlyFile) Write(data []byte, off int64) (written uint32, code fuse.Status) {
	return 0, fuse.EPERM
}

func (f *ReadOnlyFile) Flush() fuse.Status {
	return fuse.OK
}

func (f *ReadOnlyFile) Fsync(flags int) (code fuse.Status) {
	return fuse.OK
}

func (f *ReadOnlyFile) Truncate(size uint64) fuse.Status {
	return fuse.EPERM
}
