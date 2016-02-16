package main

// Note well: for now this is just a scratch program used for testing. It
// doesn't do anything useful and before it does its interface is likely
// to change significantly.

import (
	"flag"
	"fmt"
	"os"
	"time"

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
		VolumeID:          0xdeadbeef,
		Label:             [11]byte{
			'T', 'E', 'S', 'T', ' ', ' ', ' ', ' ', ' ', ' ', ' ',
		},
		ExtraClusterCount: 20,

		RootDir: &vfat.Directory{
			Dirs: []vfat.DirEntryDir{
				{
					DirEntryCommon: vfat.DirEntryCommon{
						Name:             "foobaz",
						CreationTime:     time.Now(),
						LastAccessedTime: time.Now(),
						LastModifiedTime: time.Now(),
					},
					Directory: &vfat.Directory{
						Dirs: []vfat.DirEntryDir{},
						Files: []vfat.DirEntryFile{},
					},
				},
			},
			Files: []vfat.DirEntryFile{
				{
					DirEntryCommon: vfat.DirEntryCommon{
						Name:             "hello.txt",
						CreationTime:     time.Now(),
						LastAccessedTime: time.Now(),
						LastModifiedTime: time.Now(),
					},
					BodyBuilder: &fsutil.BufferRegionBuilder{
						Buffer: []byte("Hello, world!"),
					},
				},
			},
		},
	}

	return fsutil.BuildFile(targetFn, fs)
}
