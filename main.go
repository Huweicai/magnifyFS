package main

import (
	"flag"
	"os"
	"path/filepath"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/Huweicai/magnifyFS/magnifyfs"
	"github.com/sirupsen/logrus"
)

var progName = filepath.Base(os.Args[0])

var (
	coefficient float64
	realPath    string
	mountPath   string
	debug       bool
)

func init() {
	flag.Float64Var(&coefficient, "coefficient", 1, "The magnification coefficient, will be multiplied to the raw file size.")
	flag.StringVar(&realPath, "realpath", "", "Target real directory path.")
	flag.StringVar(&mountPath, "mountpath", "", "The directory will be mounted on.")
	flag.BoolVar(&debug, "debug", false, "Show verbose detailed information.")
}

func main() {
	flag.Parse()
	if realPath == "" || mountPath == "" {
		logrus.Fatal("parameter 'realpath' or 'mountpath' is required")
	}

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if err := mount(realPath, mountPath); err != nil {
		logrus.Fatal("mount failed: %v", err)
	}
}

func mount(realPath, mountPath string) error {
	c, err := fuse.Mount(mountPath)
	if err != nil {
		return err
	}
	defer c.Close()
	logrus.Infof("fuse mounted on %s\n", mountPath)

	mfs := &magnifyfs.FS{
		RealPath:    realPath,
		Coefficient: coefficient,
	}
	if err := fs.Serve(c, mfs); err != nil {
		return err
	}

	return nil
}
