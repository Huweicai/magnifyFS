package magnifyfs

import (
	"context"
	"errors"
	"io"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/sirupsen/logrus"
)

var (
	_ = fs.Node(&File{})
	_ = fs.HandleReader(&File{})
	_ = fs.HandleWriter(&File{})
	_ = fs.NodeOpener(&File{})
	_ = fs.Node(&File{})
)

type File struct {
	fs       *FS
	realPath string
	realFile *os.File
}

func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	logrus.Debugf("File.Write called: %s size: %d offset: %d\n", f.realPath, len(req.Data), req.Offset)

	n, err := f.realFile.WriteAt(req.Data, req.Offset)
	if err != nil {
		return err
	}

	resp.Size = n
	return nil
}

func (f *File) Attr(ctx context.Context, attr *fuse.Attr) error {
	logrus.Debugf("File.Attr called: %s\n", f.realPath)

	info, err := os.Stat(f.realPath)
	if err != nil {
		return err
	}

	return f.fs.fileAttr(info, attr)
}

func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	logrus.Debugf("File.Read called: %s size: %d offset: %d\n", f.realPath, req.Size, req.Offset)

	buf := make([]byte, req.Size)
	n, err := f.realFile.ReadAt(buf, req.Offset)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	resp.Data = buf[:n]
	return nil
}

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	logrus.Debugf("File.Open called: %s %d\n", f.realPath, req.Flags)

	file, err := os.OpenFile(f.realPath, int(req.Flags), 0)
	if err != nil {
		return nil, err
	}

	f.realFile = file

	return f, err
}

func (f *File) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	logrus.Debugf("File.Release called: %s\n", f.realPath)

	return f.realFile.Close()
}
