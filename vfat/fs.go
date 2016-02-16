package vfat

import (
	"fmt"

	"golang.org/x/text/encoding/unicode"

	"github.com/apparentlymart/go-fsutil/fsutil"
)

var BasicSignature = []byte{
	0xeb, 0x58, 0x90, 'M', 'S', 'W', 'I', 'N', '4', '.', '1',
}

const ExtSignature = uint8(0x29)
const BootableSignature = uint16(0xaa55)

var FSTypeSignature = []byte("FAT32   ")

var FSInfoSignature1 = []byte{0x52, 0x52, 0x61, 0x41}
var FSInfoSignature2 = []byte{0x72, 0x72, 0x41, 0x61}
var FSInfoSignature3 = []byte{0x00, 0x00, 0x55, 0xaa}

const FATID = 0x0ffffff8
const EndOfChain uint32 = 0x0fffffff

const sectorSize = 512
const clusterSize = 4096
const fatEntrySize = 4
const sectorsPerCluster = clusterSize / sectorSize

// Two reserved sectors:
// - Boot record
// - FSInfo
const reservedSectors = 2

var LFNPadding = []byte{
	0xff, 0xff,
	0xff, 0xff,
	0xff, 0xff,
	0xff, 0xff,
	0xff, 0xff,
	0xff, 0xff,
	0xff, 0xff,
	0xff, 0xff,
	0xff, 0xff,
	0xff, 0xff,
	0xff, 0xff,
	0xff, 0xff,
	0xff, 0xff,
}

type Filesystem struct {
	HiddenSectorCount uint32
	VolumeID          uint32
	Label             [11]byte
	ExtraClusterCount uint32

	RootDir *Directory
}

type layout struct {
	DataClusters     uint32
	FATSize          uint32
	OverheadSize     uint32
	OverheadClusters uint32
	TotalClusters    uint32
}

func (fs *Filesystem) calcLayout() *layout {
	reservedSize := uint32(reservedSectors * sectorSize)
	dataClusters := uint32(fs.RootDir.TotalClusters(true))

	fatSize := uint32(dataClusters * fatEntrySize)

	overheadSize := reservedSize + fatSize
	if overheadSize < (2 * clusterSize) {
		// Must always have at least two clusters of overhead because
		// the first two entries in the FAT are used for metadata.
		overheadSize = 2 * clusterSize
	}

	overheadClusters := divCeil(overheadSize, clusterSize)

	// Do we have so much overhead that it requires extra overhead clusters?
	if overheadClusters > 2 {
		fatSize += (overheadClusters - 2) * fatEntrySize
		overheadSize = reservedSize + fatSize
		overheadClusters = divCeil(overheadSize, clusterSize)
	}

	totalClusters := overheadClusters + dataClusters + fs.ExtraClusterCount
	fatSize = totalClusters * fatEntrySize

	return &layout{
		DataClusters:     dataClusters,
		FATSize:          fatSize,
		OverheadSize:     overheadSize,
		OverheadClusters: overheadClusters,
		TotalClusters:    totalClusters,
	}
}

func (fs *Filesystem) Length() int {
	layout := fs.calcLayout()
	return int(layout.TotalClusters * clusterSize)
}

func (fs *Filesystem) Build(region fsutil.Region) {
	bootRecord := region.Slice(0, sectorSize)

	layout := fs.calcLayout()
	sectorsPerFAT := divCeil(layout.FATSize, sectorSize)
	nextCluster := uint32(layout.OverheadClusters)
	totalSectors := uint32(layout.TotalClusters * sectorsPerCluster)
	if nextCluster < 2 {
		// Data can't occupy clusters 0 or 1 because the FAT entries
		// for these clusters are used for other purposes.
		nextCluster = 2
	}

	// Main Signatures
	bootRecord.WriteBytes(0, BasicSignature)
	bootRecord.WriteU16LE(0x1fe, BootableSignature)

	// BIOS Parameter Block
	bootRecord.WriteU16LE(0x00b, sectorSize)
	bootRecord.WriteU8(0x00d, sectorsPerCluster)
	bootRecord.WriteU16LE(0x00e, reservedSectors)
	bootRecord.WriteU8(0x010, 1)     // Number of FATs
	bootRecord.WriteU16LE(0x011, 0)  // Number of root entries not used on FAT32
	bootRecord.WriteU8(0x015, 0xf8)  // Media Descriptor (Fixed Disk)
	bootRecord.WriteU16LE(0x018, 1)  // Physical sectors per track not used
	bootRecord.WriteU16LE(0x01a, 64) // Number of heads not used
	bootRecord.WriteU32LE(0x020, totalSectors)
	bootRecord.WriteU16LE(0x02a, 0) // Version number
	bootRecord.WriteU32LE(0x024, sectorsPerFAT)
	bootRecord.WriteU16LE(0x030, 1) // Sector of FSInfo
	bootRecord.WriteU8(0x042, ExtSignature)
	bootRecord.WriteU32LE(0x043, fs.VolumeID)
	bootRecord.WriteBytes(0x047, fs.Label[:])
	bootRecord.WriteBytes(0x052, FSTypeSignature)

	fsInfo := region.Slice(sectorSize, sectorSize)
	fsInfo.WriteBytes(0x000, FSInfoSignature1)
	fsInfo.WriteBytes(0x1e4, FSInfoSignature2)
	fsInfo.WriteU32LE(0x1e8, 0xffffffff) // Free data clusters not known yet
	fsInfo.WriteU32LE(0x1ec, 0xffffffff) // No most recent data cluster
	fsInfo.WriteBytes(0x1fc, FSInfoSignature3)

	fat := region.Slice(reservedSectors*sectorSize, int(layout.FATSize))
	fat.WriteU32LE(0, FATID)
	fat.WriteU32LE(4, EndOfChain) // End of chain marker used elsewhere in FAT

	// Now we'll walk the caller's provided directory tree and produce
	// the actual filesystem data.

	lfnEncoding := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	lfnEncoder := lfnEncoding.NewEncoder()

	allocCluster := func() uint32 {
		nextCluster += 1
		return nextCluster - 1
	}

	// Writes a directory and returns the cluster where it begins
	var writeDirectory func(*Directory, bool) uint32
	writeDirectory = func(dir *Directory, isRoot bool) uint32 {
		startCluster := allocCluster()

		tableBytes := uint32(dir.TableBytes(isRoot))
		tableClusterCount := divCeil(tableBytes, clusterSize)
		currentCluster := startCluster

		for i := uint32(1); i < tableClusterCount; i += 1 {
			nextCluster := allocCluster()
			// Write next cluster number into the FAT entry for the
			// current cluster, creating a chain.

			fat.WriteU32LE(int(currentCluster*4), nextCluster)

			currentCluster = nextCluster
		}

		// Now write the "End of chain" marker into the FAT entry for
		// our final cluster.
		fat.WriteU32LE(int(currentCluster*4), EndOfChain)

		// We guarantee that the directory table gets allocated consecutive
		// clusters, so we can just create a flat sub-region for it.
		tableRegion := region.Slice(int(startCluster*clusterSize), int(tableClusterCount*clusterSize))

		entryOffset := 0

		if isRoot {
			// Special entry for the volume label
			tableRegion.WriteBytes(0x00, fs.Label[:])
			tableRegion.WriteU8(0x0b, byte(VolumeIDAttr))
			entryOffset += DirEntrySize
		}

		// We always visit directories first since that causes all of the
		// directory tables to be kept together at the start of the filesystem
		// and thus we maximize locality for path traversal.
		// However, a side-effect of this is that the root directory files
		// will be very far away from their directory entries. Might revisit
		// this strategy later.

		writeLFN := func (entry DirEntryCommon, dosFN []byte) {
			lfn, err := lfnEncoder.Bytes([]byte(entry.Name))
			if err != nil {
				panic(err)
			}

			checksum := byte(0)
			for _, b := range dosFN {
				checksum = ((checksum & 1) << 7) + (checksum >> 1) + b
			}

			idx := byte(1)
			for {
				entryRegion := tableRegion.Slice(entryOffset, DirEntrySize)
				entryOffset += DirEntrySize

				entryRegion.WriteU8(0x00, idx)
				idx += 1

				entryRegion.WriteU8(0x0b, byte(LFNAttrs))
				entryRegion.WriteU8(0x0d, checksum)

				// Write the padding in and then we'll write the actual
				// date over the top.
				entryRegion.WriteBytes(0x01, LFNPadding[0:10])
				entryRegion.WriteBytes(0x12, LFNPadding[0:12])
				entryRegion.WriteBytes(0x1c, LFNPadding[0:4])

				toCopy := lfn
				if len(toCopy) > 10 {
					toCopy = toCopy[:10]
				}
				copied := entryRegion.WriteBytes(0x01, toCopy)
				if copied < 10 {
					entryRegion.WriteU16LE(0x01 + copied, 0)
					break
				}
				lfn = lfn[10:]

				toCopy = lfn
				if len(toCopy) > 12 {
					toCopy = toCopy[:12]
				}
				copied = entryRegion.WriteBytes(0x0e, toCopy)
				if copied < 12 {
					entryRegion.WriteU16LE(0x0e + copied, 0)
					break
				}
				lfn = lfn[12:]

				toCopy = lfn
				if len(toCopy) > 4 {
					toCopy = toCopy[:4]
				}
				copied = entryRegion.WriteBytes(0x1c, toCopy)
				if copied < 4 {
					entryRegion.WriteU16LE(0x1c + copied, 0)
					break
				}
				lfn = lfn[4:]
			}
		}

		for _, entry := range dir.Dirs {
			startCluster := writeDirectory(entry.Directory, false)

			// For now we just use junk short filenames, since no reasonable
			// OS looks at these anymore anyway.
			dosFN := []byte(fmt.Sprintf("%08xLFN", startCluster))

			writeLFN(entry.DirEntryCommon, dosFN)
			entryRegion := tableRegion.Slice(entryOffset, DirEntrySize)
			entryOffset += DirEntrySize

			entryRegion.WriteBytes(0x00, dosFN)
			entryRegion.WriteU8(0x0b, byte(entry.Attributes | DirectoryAttr))
			entryRegion.WriteU16LE(0x14, uint16(startCluster >> 16))
			entryRegion.WriteU16LE(0x1a, uint16(startCluster))
		}

		return startCluster
	}

	// Always start with the root directory
	rootDirCluster := writeDirectory(fs.RootDir, true)
	bootRecord.WriteU32LE(0x02c, rootDirCluster)
}

func divCeil(a uint32, b uint32) uint32 {
	if (a % b) != 0 {
		return (a / b) + 1
	} else {
		return a / b
	}
}
