package magnifyfs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/sirupsen/logrus"
)

// Directory represents a file directory
type Directory struct {
	fs       *FS
	realPath string
}

var (
	_ = fs.Node(&Directory{})
	_ = fs.NodeRequestLookuper(&Directory{})
	_ = fs.HandleReadDirAller(&Directory{})
	_ = fs.NodeCreater(&Directory{})
)

func (d *Directory) Attr(ctx context.Context, attr *fuse.Attr) error {
	logrus.Debugf("Directory.Attr called: %s\n", d.realPath)

	info, err := os.Stat(d.realPath)
	if err != nil {
		return err
	}

	return d.fs.fileAttr(info, attr)
}

func (f FS) fileAttr(info os.FileInfo, attr *fuse.Attr) error {
	sysStat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("unexpected type of FileInfo.Sys")
	}

	attr.Inode = sysStat.Ino
	if info.IsDir() {
		attr.Size = uint64(info.Size())
	} else {
		attr.Size = uint64(float64(info.Size()) * f.Coefficient)
	}

	attr.Mode = info.Mode()
	attr.Mtime = info.ModTime()
	attr.Ctime = time.Unix(sysStat.Ctim.Unix())

	return nil
}

func (d *Directory) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	logrus.Debugf("Directory.Lookup called: %s %s\n", d.realPath, req.Name)

	return d.lookup(req.Name)
}

func (d *Directory) lookup(name string) (fs.Node, error) {
	realPath := filepath.Join(d.realPath, name)
	realFileInfo, err := os.Stat(realPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, syscall.ENOENT
		}

		return nil, err
	}

	if realFileInfo.IsDir() {
		return &Directory{
			fs:       d.fs,
			realPath: realPath,
		}, nil
	}

	return &File{
		fs:       d.fs,
		realPath: realPath,
	}, err
}

// ReadDirAll is used to list all items under a directory, called such as 'ls'
func (d *Directory) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	logrus.Debugf("Directory.ReadDirAll called: %s\n", d.realPath)

	entries, err := os.ReadDir(d.realPath)
	if err != nil {
		return nil, err
	}

	fuseEntries := make([]fuse.Dirent, len(entries))
	for i, entry := range entries {
		fuseEntry, err := osEntryToFuseEntry(entry)
		if err != nil {
			return nil, err
		}

		fuseEntries[i] = fuseEntry
	}

	return fuseEntries, nil
}

func osEntryToFuseEntry(entry os.DirEntry) (fuse.Dirent, error) {
	info, err := entry.Info()
	if err != nil {
		return fuse.Dirent{}, err
	}

	sysStat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fuse.Dirent{}, errors.New("unexpected type of FileInfo.Sys")
	}

	// todo support other type
	tp := fuse.DT_File
	if info.Mode().IsDir() {
		tp = fuse.DT_Dir
	}

	return fuse.Dirent{
		Inode: sysStat.Ino,
		Type:  tp,
		Name:  entry.Name(),
	}, nil
}

func (d *Directory) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	realPath := filepath.Join(d.realPath, req.Name)
	logrus.Debugf("Directory.Remove called: %s\n", realPath)

	return os.Remove(realPath)
}

func (d *Directory) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	new, ok := newDir.(*Directory)
	if !ok {
		return errors.New("unexpected directory node type")
	}

	oldPath := filepath.Join(d.realPath, req.OldName)
	newPath := filepath.Join(new.realPath, req.NewName)
	logrus.Debugf("Directory.Rename called: %s -> %s\n", oldPath, newPath)

	// todo solve the cache node
	return os.Rename(oldPath, newPath)
}

func (d *Directory) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	realPath := filepath.Join(d.realPath, req.Name)
	logrus.Debugf("Directory.Create called: %s\n", realPath)

	realFile, err := os.OpenFile(req.Name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, req.Mode.Perm())
	if err != nil {
		return nil, nil, err
	}

	file := &File{
		fs:       d.fs,
		realPath: realPath,
		realFile: realFile,
	}

	return file, file, nil
}
