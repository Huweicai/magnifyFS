package magnifyfs

import (
	"bazil.org/fuse/fs"
)

type FS struct {
	Coefficient float64
	RealPath    string
}

func (f FS) Root() (fs.Node, error) {
	return &Directory{
		fs:       &f,
		realPath: f.RealPath,
	}, nil
}
