//+build windows

package local

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

var (
	ntdll                            = windows.NewLazySystemDLL("ntdll.dll")
	procNtQueryVolumeInformationFile = ntdll.NewProc("NtQueryVolumeInformationFile")
	procNtSetInformationFileProc     = ntdll.NewProc("NtSetInformationFile")
)

type fileAllocationInformation struct {
	AllocationSize uint64
}

type fileFsSizeInformation struct {
	TotalAllocationUnits     uint64
	AvailableAllocationUnits uint64
	SectorsPerAllocationUnit uint32
	BytesPerSector           uint32
}

type ioStatusBlock struct {
	Status, Information uintptr
}

func ntQueryVolumeInformationFile(handle uintptr, iosb *ioStatusBlock, information uintptr, length uint32, class uint32) (err error) {
	_, _, e1 := syscall.Syscall6(procNtQueryVolumeInformationFile.Addr(), 5, uintptr(handle), uintptr(unsafe.Pointer(iosb)), uintptr(information), uintptr(length), uintptr(class), 0)
	if e1 == 0 {
		return nil
	}
	return e1
}

func ntSetInformationFile(handle uintptr, iosb *ioStatusBlock, information uintptr, length uint32, class uint32) (err error) {
	_, _, e1 := syscall.Syscall6(procNtSetInformationFileProc.Addr(), 5, uintptr(handle), uintptr(unsafe.Pointer(iosb)), uintptr(information), uintptr(length), uintptr(class), 0)
	if e1 == 0 {
		return nil
	}
	return e1
}

// preAllocate the file for performance reasons
func preAllocate(size int64, out *os.File) error {
	if size <= 0 {
		return nil
	}
	var (
		iosb       ioStatusBlock
		fsSizeInfo fileFsSizeInformation
		allocInfo  fileAllocationInformation
	)
	err := ntQueryVolumeInformationFile(out.Fd(), &iosb, uintptr(unsafe.Pointer(&fsSizeInfo)), uint32(unsafe.Sizeof(fsSizeInfo)), 3)
	if err != nil {
		return errors.Wrap(err, "preAllocate NtQueryVolumeInformationFile failed")
	}
	clusterSize := uint64(fsSizeInfo.BytesPerSector) * uint64(fsSizeInfo.SectorsPerAllocationUnit)
	allocInfo.AllocationSize = (1 + uint64(size-1)/clusterSize) * clusterSize
	err = ntSetInformationFile(out.Fd(), &iosb, uintptr(unsafe.Pointer(&allocInfo)), uint32(unsafe.Sizeof(allocInfo)), 19)
	if err != nil {
		return errors.Wrap(err, "preAllocate NtSetInformationFile failed")
	}
	return nil
}
