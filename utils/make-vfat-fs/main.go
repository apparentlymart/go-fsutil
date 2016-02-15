package main

// Note well: for now this is just a scratch program used for testing. It
// doesn't do anything useful and before it does its interface is likely
// to change significantly.

import (
	"flag"
	"fmt"
	"os"

	"github.com/apparentlymart/go-fsutil/fsutil"
	"github.com/apparentlymart/go-fsutil/vfat"
)

func main() {
	flag.Parse()

	targetFn := flag.Arg(0)

	err := run(targetFn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(2)
	}
}

func run(targetFn string) error {
	fs := &vfat.Filesystem{
		RootDir: &vfat.Directory{
			Dirs:  []vfat.DirEntryDir{},
			Files: []vfat.DirEntryFile{},
		},
	}

	return fsutil.BuildFile(targetFn, fs)
}
