package main

import (
	"context"
	"log"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type UnixFSNode struct {
	nodefs.Node
	Path string
}

func (n *UnixFSNode) Lookup(out *fuse.Attr, name string, ctx *fuse.Context) (*nodefs.Inode, fuse.Status) {
	childPath := path.Join(n.Path, name)
	stat, err := FastStat(context.TODO(), childPath)
	if err != nil {
		log.Println("Lookup", childPath, err)
		return nil, fuse.EIO
	}
	if stat == nil {
		n.Inode().RmChild(name)
		return nil, fuse.ENOENT
	}

	out.Mtime = 1
	out.Ctime = 1
	out.Size = stat.Size
	out.Blocks = out.Size
	out.Blksize = 1
	out.Mode = 0644
	if stat.Type == "directory" {
		out.Mode |= fuse.S_IFDIR | 0111
	} else {
		out.Mode |= fuse.S_IFREG
	}

	node := &UnixFSNode{
		Node: nodefs.NewDefaultNode(),
		Path: childPath,
	}

	return n.Inode().NewChild(name, out.IsDir(), node), fuse.OK
}

func (n *UnixFSNode) OpenDir(ctx *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	list, err := FastList(context.TODO(), n.Path+"/")
	if err != nil {
		log.Println("OpenDir", n.Path, err)
		return nil, fuse.EIO
	}
	if list == nil {
		return nil, fuse.ENOENT
	}
	if len(list.Entries) == 1 && list.Entries[0].Name == "" {
		return nil, fuse.ENOTDIR
	}

	if !n.Inode().IsDir() {
		p, name := n.Inode().Parent()
		p.RmChild(name)
		n.SetInode(p.NewChild(name, true, n))
	}

	existing := n.Inode().Children()
	for _, entry := range list.Entries {
		isDir := entry.Type == Directory
		if e, ok := existing[entry.Name]; ok {
			delete(existing, entry.Name)

			if e.IsDir() == isDir {
				continue
			}

			n.Inode().RmChild(entry.Name)
		}

		n.Inode().NewChild(entry.Name, isDir, &UnixFSNode{
			Node: nodefs.NewDefaultNode(),
			Path: path.Join(n.Path, entry.Name),
		})
	}

	if n.Path == "/" {
		delete(existing, "ipfs")
		delete(existing, "ipns")
	}

	for name := range existing {
		n.Inode().RmChild(name)
	}

	return n.Node.OpenDir(ctx)
}

func (n *UnixFSNode) GetAttr(out *fuse.Attr, file nodefs.File, ctx *fuse.Context) fuse.Status {
	stat, err := FastStat(context.TODO(), n.Path)
	if err != nil {
		log.Println("GetAttr", n.Path, err)
		return fuse.EIO
	}
	if stat == nil {
		return fuse.ENOENT
	}

	out.Mtime = 1
	out.Ctime = 1
	out.Size = stat.Size
	out.Mode = 0644
	if stat.Type == "directory" {
		out.Mode |= fuse.S_IFDIR | 0111
	} else {
		out.Mode |= fuse.S_IFREG
	}

	return fuse.OK
}

func (n *UnixFSNode) GetXAttr(attribute string, ctx *fuse.Context) ([]byte, fuse.Status) {
	switch attribute {
	case "user.ipfs-hash":
		stat, err := Stat(context.TODO(), n.Path)
		if err != nil {
			log.Println("GetXAttr", n.Path, err)
			return nil, fuse.EIO
		}
		if stat == nil {
			return nil, fuse.ENOENT
		}

		return []byte(stat.Hash), fuse.OK
	default:
		return nil, fuse.ENOATTR
	}
}
func (n *UnixFSNode) RemoveXAttr(attr string, ctx *fuse.Context) fuse.Status {
	return fuse.EPERM
}
func (n *UnixFSNode) SetXAttr(attr string, data []byte, flags int, ctx *fuse.Context) fuse.Status {
	return fuse.EPERM
}
func (n *UnixFSNode) ListXAttr(ctx *fuse.Context) ([]string, fuse.Status) {
	return []string{"user.ipfs-hash"}, fuse.OK
}

func (n *UnixFSNode) Mkdir(name string, mode uint32, ctx *fuse.Context) (*nodefs.Inode, fuse.Status) {
	dirName := path.Join(n.Path, name)
	resp, err := ipfs.Request("files/mkdir", dirName).Send(context.TODO())
	if err != nil {
		log.Println("Mkdir", dirName, err)
		return nil, fuse.EIO
	}
	if err = resp.Close(); err != nil {
		log.Println("Mkdir", dirName, err)
		return nil, fuse.EIO
	}
	if resp.Error != nil {
		switch resp.Error.Message {
		case "file does not exist":
			return nil, fuse.ENOENT
		case "file already exists":
			return nil, fuse.Status(syscall.EEXIST)
		default:
			log.Println("Mkdir", dirName, resp.Error)
			return nil, fuse.EIO
		}
	}

	return n.Inode().NewChild(name, true, &UnixFSNode{
		Node: nodefs.NewDefaultNode(),
		Path: dirName,
	}), fuse.OK
}

func (n *UnixFSNode) Unlink(name string, ctx *fuse.Context) fuse.Status {
	childPath := path.Join(n.Path, name)
	resp, err := ipfs.Request("files/rm", childPath).Send(context.TODO())
	if err == nil {
		err = resp.Close()
	}
	if err == nil && resp.Error != nil {
		if strings.HasSuffix(resp.Error.Message, " is a directory, use -r to remove directories") {
			return fuse.EISDIR
		} else if resp.Error.Message == "file does not exist" {
			n.Inode().RmChild(name)
			return fuse.ENOENT
		}

		err = resp.Error
	}

	if err != nil {
		log.Println("Unlink", childPath, err)
		return fuse.EIO
	}

	n.Inode().RmChild(name)
	return fuse.OK
}
func (n *UnixFSNode) Rmdir(name string, ctx *fuse.Context) fuse.Status {
	childPath := path.Join(n.Path, name)
	resp, err := ipfs.Request("files/rm", childPath).Option("recursive", true).Send(context.TODO())
	if err == nil {
		err = resp.Close()
	}
	if err == nil && resp.Error != nil {
		if resp.Error.Message == "file does not exist" {
			n.Inode().RmChild(name)
			return fuse.ENOENT
		}

		err = resp.Error
	}

	if err != nil {
		log.Println("Rmdir", childPath, err)
		return fuse.EIO
	}

	n.Inode().RmChild(name)
	return fuse.OK
}

func (n *UnixFSNode) Rename(oldName string, newParent nodefs.Node, newName string, ctx *fuse.Context) fuse.Status {
	if root, ok := newParent.(*UnixFSRootNode); ok {
		newParent = &root.UnixFSNode
	}
	if np, ok := newParent.(*UnixFSNode); ok {
		oldPath := path.Join(n.Path, oldName)
		newPath := path.Join(np.Path, newName)

		resp, err := ipfs.Request("files/mv", oldPath, newPath).Send(context.TODO())
		if err == nil {
			err = resp.Close()
		}
		if err == nil && resp.Error != nil {
			err = resp.Error
		}

		if err != nil {
			log.Println("Rename", oldPath, newPath, err)
			return fuse.EIO
		}

		n.Inode().RmChild(oldName)
		return fuse.OK
	}
	return fuse.EPERM
}

func (n *UnixFSNode) Create(name string, flags uint32, mode uint32, ctx *fuse.Context) (nodefs.File, *nodefs.Inode, fuse.Status) {
	inode, status := n.Mknod(name, mode, 0, ctx)
	if status != fuse.OK {
		return nil, inode, status
	}

	node := inode.Node().(*UnixFSNode)
	return &nodefs.WithFlags{
		Description: node.Path,
		File: &UnixFSFile{
			File: nodefs.NewDefaultFile(),
			Node: node,
		},
		OpenFlags: flags,
	}, inode, fuse.OK
}
func (n *UnixFSNode) Open(flags uint32, ctx *fuse.Context) (nodefs.File, fuse.Status) {
	return &nodefs.WithFlags{
		Description: n.Path,
		File: &UnixFSFile{
			File: nodefs.NewDefaultFile(),
			Node: n,
		},
		OpenFlags: flags,
	}, fuse.OK
}

func (n *UnixFSNode) Mknod(name string, mode uint32, dev uint32, ctx *fuse.Context) (*nodefs.Inode, fuse.Status) {
	if dev != 0 {
		// only allow regular files
		return nil, fuse.ENODEV
	}
	if mode&^0777 == fuse.S_IFDIR {
		return n.Mkdir(name, mode, ctx)
	}
	if mode&^0777 != fuse.S_IFREG {
		return nil, fuse.EINVAL
	}

	childPath := path.Join(n.Path, name)
	resp, err := attachFile(ipfs.Request("files/write", childPath), nil).Option("create", true).Option("raw-leaves", true).Option("flush", false).Send(context.TODO())
	if err == nil {
		err = resp.Close()
	}
	if err == nil && resp.Error != nil {
		if strings.HasSuffix(resp.Error.Message, " was not a file") {
			return nil, fuse.Status(syscall.EEXIST)
		}
		err = resp.Error
	}
	if err != nil {
		log.Println("Mknod", childPath, err)
		return nil, fuse.EIO
	}

	return n.Inode().NewChild(name, false, &UnixFSNode{
		Node: nodefs.NewDefaultNode(),
		Path: childPath,
	}), fuse.OK
}

func (n *UnixFSNode) Utimens(file nodefs.File, atime *time.Time, mtime *time.Time, ctx *fuse.Context) fuse.Status {
	// We don't have timestamps, so just ignore the system call.
	return fuse.OK
}
