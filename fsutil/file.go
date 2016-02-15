package fsutil

import (
	"os"

	"github.com/edsrzf/mmap-go"
)

type Protection int

const (
	ReadOnly    Protection = mmap.RDONLY
	ReadWrite   Protection = mmap.RDWR
	CopyOnWrite Protection = mmap.COPY
	Executable  Protection = mmap.EXEC
)

type RegionFile struct {
	Region Region
}

func (rf *RegionFile) Close() error {
	return (*mmap.MMap)(&(rf.Region[0])).Unmap()
}

func RegionForFile(f *os.File, prot Protection) (RegionFile, error) {
	buf, err := mmap.Map(f, int(prot), 0)
	if err != nil {
		return RegionFile{}, err
	}

	return RegionFile{
		RegionForBytes([]byte(buf)),
	}, nil
}

func CreateFile(fn string, size int) (RegionFile, error) {
	f, err := os.Create(fn)
	if err != nil {
		return RegionFile{}, err
	}

	err = f.Truncate(int64(size))
	if err != nil {
		return RegionFile{}, err
	}

	return RegionForFile(f, ReadWrite)
}

func OpenFile(fn string, prot Protection) (RegionFile, error) {
	f, err := os.Open(fn)
	if err != nil {
		return RegionFile{}, err
	}

	return RegionForFile(f, prot)
}

func BuildFile(fn string, builder RegionBuilder) error {
	size := builder.Length()

	rf, err := CreateFile(fn, size)
	if err != nil {
		return err
	}

	builder.Build(rf.Region)

	return rf.Close()
}
