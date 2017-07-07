// +build windows
// FIXME Not tested yet!
package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	Modkernel32           = windows.NewLazyDLL("kernel32.dll")
	procGetDiskFreeSpaceW = Modkernel32.NewProc("GetDiskFreeSpaceW")
)

func statfs(path string) int64 {
	lpSectorsPerCluster := uint32(0)
	lpBytesPerSector := uint32(0)
	lpNumberOfFreeClusters := uint32(0)
	lpTotalNumberOfClusters := uint32(0)
	diskret, _, _ := procGetDiskFreeSpaceW.Call(
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(path))),
		uintptr(unsafe.Pointer(&lpSectorsPerCluster)),
		uintptr(unsafe.Pointer(&lpBytesPerSector)),
		uintptr(unsafe.Pointer(&lpNumberOfFreeClusters)),
		uintptr(unsafe.Pointer(&lpTotalNumberOfClusters)),
	)
	if diskret == 0 {
		return 4096
	}
	return int64(lpBytesPerSector)
}
