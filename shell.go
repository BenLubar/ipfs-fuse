package main

import (
	"bytes"
	"context"
	"mime/multipart"
	pathutil "path"

	shell "github.com/ipfs/go-ipfs-api"
)

var ipfs = shell.NewLocalShell()

type NodeType int

const (
	Directory NodeType = 0
	File      NodeType = 1
)

type UnixFSList struct {
	Entries []UnixFSDirEntry
}

type UnixFSDirEntry struct {
	Name string
	Type NodeType
	Size uint64
	Hash string
}

type UnixFSStat struct {
	Hash           string
	Size           uint64
	CumulativeSize uint64
	Blocks         int
	Type           string
	WithLocality   bool
	Local          bool
	SizeLocal      uint64
}

func Stat(ctx context.Context, path string) (*UnixFSStat, error) {
	var data UnixFSStat
	if err := ipfs.Request("files/stat", path).Option("flush", false).Exec(ctx, &data); err != nil {
		if ie, ok := err.(*shell.Error); ok && ie.Message == "file does not exist" {
			// Not Found
			return nil, nil
		}

		return nil, err
	}
	return &data, nil
}

func List(ctx context.Context, path string, long bool) (*UnixFSList, error) {
	var data UnixFSList
	if err := ipfs.Request("files/ls", path).Option("flush", false).Option("l", long).Exec(ctx, &data); err != nil {
		if ie, ok := err.(*shell.Error); ok && ie.Message == "file does not exist" {
			// Not Found
			return nil, nil
		}

		return nil, err
	}
	return &data, nil
}

func FastStat(ctx context.Context, path string) (*UnixFSStat, error) {
	// Stat-ing a directory is very slow.

	// First thing to try: List it as if it were a directory.
	list, err := List(ctx, path+"/", false)
	if err != nil || list == nil {
		return nil, err
	}

	// The only entry is an unnamed file.
	if len(list.Entries) == 1 && list.Entries[0].Name == "" {
		// It is safe to call Stat on files for the most part.
		return Stat(ctx, path)
	}

	return &UnixFSStat{
		Type: "directory",
		Hash: "", // hash not available through this method
		Size: uint64(len(list.Entries)),
	}, nil
}

func FastList(ctx context.Context, path string) (*UnixFSList, error) {
	list, err := List(ctx, path+"/", false)
	if err != nil || list == nil || len(list.Entries) == 0 ||
		(len(list.Entries) == 1 && list.Entries[0].Name == "") {
		return list, err
	}

	if len(list.Entries) > 100 {
		// Bite the bullet and just go for the slow route.
		return List(ctx, path+"/", true)
	}

	for _, entry := range list.Entries {
		stat, err := FastStat(ctx, pathutil.Join(path, entry.Name))
		if err != nil || stat == nil {
			return nil, err
		}

		if stat.Type == "directory" {
			entry.Type = Directory
		} else {
			entry.Type = File
		}

		entry.Hash = stat.Hash
		entry.Size = stat.Size
	}

	return list, nil
}

func ListImmutable(ctx context.Context, path string) (*UnixFSList, error) {
	var data struct {
		Objects []shell.LsObject
	}
	if err := ipfs.Request("ls", path).Exec(ctx, &data); err != nil {
		if ie, ok := err.(*shell.Error); ok && ie.Message == "file does not exist" {
			// Not Found
			return nil, nil
		}
		return nil, err
	}

	links := data.Objects[0].Links
	var list UnixFSList
	for _, l := range links {
		var t NodeType
		switch l.Type {
		case shell.TDirectory:
			t = Directory
		default:
			t = File
		}
		list.Entries = append(list.Entries, UnixFSDirEntry{
			Hash: l.Hash,
			Name: l.Name,
			Size: l.Size,
			Type: t,
		})
	}

	return &list, nil
}

func Resolve(ctx context.Context, name string) (string, error) {
	var data struct {
		Path string
	}
	if err := ipfs.Request("resolve", "/ipns/"+name).Exec(ctx, &data); err != nil {
		if ie, ok := err.(*shell.Error); ok && ie.Message == "file does not exist" {
			// Not Found
			return "", nil
		}
		return "", err
	}
	return data.Path, nil
}

func attachFile(builder *shell.RequestBuilder, data []byte) *shell.RequestBuilder {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	pw, err := w.CreateFormFile("data", "file")
	if err != nil {
		panic(err)
	}
	_, err = pw.Write(data)
	if err != nil {
		panic(err)
	}
	err = w.Close()
	if err != nil {
		panic(err)
	}

	return builder.Body(&buf).Header("Content-Type", w.FormDataContentType())
}
