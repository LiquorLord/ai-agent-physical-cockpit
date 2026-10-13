package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"syscall"
	"unsafe"
)

type FbBitfield struct {
	Offset   uint32
	Length   uint32
	MsbRight uint32
}

type FbVarScreeninfo struct {
	Xres, Yres                                        uint32
	XresVirtual, YresVirtual                          uint32
	Xoffset, Yoffset                                  uint32
	BitsPerPixel                                      uint32
	Grayscale                                         uint32
	Red, Green, Blue, Transp                          FbBitfield
	Nonstd                                            uint32
	Activate                                          uint32
	Height, Width                                     uint32
	AccelFlags                                        uint32
	Pixclock                                          uint32
	LeftMargin, RightMargin, UpperMargin, LowerMargin uint32
	HsyncLen, VsyncLen                                uint32
	Sync, Vmode, Rotate, Colorspace                   uint32
	Reserved                                          [4]uint32
}

const (
	FBIOGET_VSCREENINFO = 0x4600
	FBIOPUT_VSCREENINFO = 0x4601
	FBIOPAN_DISPLAY     = 0x4606
	FB_ACTIVATE_NOW     = 0
	FB_ACTIVATE_FORCE   = 128
)

func main() {
	rawPath := "/mnt/sdcard/splash.raw"
	if len(os.Args) > 1 {
		rawPath = os.Args[1]
	}

	fb, err := os.OpenFile("/dev/fb0", os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("Open /dev/fb0 failed: %v", err)
	}
	defer fb.Close()

	// 1. If image file exists, write to fb0
	if rawData, err := os.ReadFile(rawPath); err == nil && len(rawData) > 0 {
		fb.Seek(0, io.SeekStart)
		fb.Write(rawData)
		fmt.Printf("[+] Loaded %d bytes from %s to /dev/fb0\n", len(rawData), rawPath)
	} else {
		fmt.Printf("[-] Could not read %s: %v\n", rawPath, err)
	}

	// 2. Trigger display pan and activation
	var v FbVarScreeninfo
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fb.Fd(), FBIOGET_VSCREENINFO, uintptr(unsafe.Pointer(&v)))
	if errno != 0 {
		log.Fatalf("GET_VSCREENINFO failed: %v", errno)
	}

	v.Activate = FB_ACTIVATE_NOW | FB_ACTIVATE_FORCE
	v.Xoffset = 0
	v.Yoffset = 0

	syscall.Syscall(syscall.SYS_IOCTL, fb.Fd(), FBIOPUT_VSCREENINFO, uintptr(unsafe.Pointer(&v)))
	syscall.Syscall(syscall.SYS_IOCTL, fb.Fd(), FBIOPAN_DISPLAY, uintptr(unsafe.Pointer(&v)))

	fmt.Printf("[+] Display activated: %dx%d (virtual %dx%d), bpp: %d, rotate: %d\n",
		v.Xres, v.Yres, v.XresVirtual, v.YresVirtual, v.BitsPerPixel, v.Rotate)
}
