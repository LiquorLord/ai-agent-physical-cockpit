package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

type v4l2_capability struct {
	driver       [16]byte
	card         [32]byte
	bus_info     [32]byte
	version      uint32
	capabilities uint32
	device_caps  uint32
	reserved     [3]uint32
}

type v4l2_plane_pix_format struct {
	sizeimage    uint32
	bytesperline uint32
	reserved     [6]uint16
}

type v4l2_pix_format_mplane struct {
	width        uint32
	height       uint32
	pixelformat  uint32
	field        uint32
	colorspace   uint32
	plane_fmt    [8]v4l2_plane_pix_format
	num_planes   uint8
	flags        uint8
	ycbcr_enc    uint8
	quantization uint8
	xfer_func    uint8
	reserved     [7]uint8
}

type v4l2_format_mp struct {
	buf_type uint32
	fmt      [200]byte // exactly 200 bytes -> total struct size 204 bytes
}

const (
	VIDIOC_QUERYCAP = 0x80685600
	VIDIOC_G_FMT    = 0xc0cc5604 // 32-bit ARM exact value (size 204)
	VIDIOC_S_FMT    = 0xc0cc5605
	V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE = 9
	V4L2_BUF_TYPE_VIDEO_CAPTURE        = 1
)

func main() {
	dev := "/dev/video0"
	if len(os.Args) > 1 {
		dev = os.Args[1]
	}

	f, err := os.OpenFile(dev, os.O_RDWR, 0)
	if err != nil {
		fmt.Printf("Open error on %s: %v\n", dev, err)
		return
	}
	defer f.Close()

	var cap v4l2_capability
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), VIDIOC_QUERYCAP, uintptr(unsafe.Pointer(&cap)))
	if errno != 0 {
		fmt.Printf("QUERYCAP error: %v\n", errno)
		return
	}

	fmt.Printf("[%s] Driver: %s, Caps: 0x%08x\n", dev, string(cap.driver[:]), cap.capabilities)

	var fmtReq v4l2_format_mp
	fmtReq.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), VIDIOC_G_FMT, uintptr(unsafe.Pointer(&fmtReq)))
	if errno != 0 {
		fmt.Printf("G_FMT error: %v\n", errno)
	} else {
		pix := (*v4l2_pix_format_mplane)(unsafe.Pointer(&fmtReq.fmt[0]))
		fourcc := string([]byte{
			byte(pix.pixelformat & 0xff),
			byte((pix.pixelformat >> 8) & 0xff),
			byte((pix.pixelformat >> 16) & 0xff),
			byte((pix.pixelformat >> 24) & 0xff),
		})
		fmt.Printf("Current Format: %dx%d, PixelFormat: %s (0x%08x), Planes: %d\n",
			pix.width, pix.height, fourcc, pix.pixelformat, pix.num_planes)
		for i := 0; i < int(pix.num_planes); i++ {
			fmt.Printf("  Plane %d: sizeimage=%d, bytesperline=%d\n",
				i, pix.plane_fmt[i].sizeimage, pix.plane_fmt[i].bytesperline)
		}
	}

	var setFmt v4l2_format_mp
	setFmt.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	spix := (*v4l2_pix_format_mplane)(unsafe.Pointer(&setFmt.fmt[0]))
	spix.width = 1280
	spix.height = 720
	spix.pixelformat = 0x3231564e // 'NV12'
	spix.num_planes = 1
	spix.plane_fmt[0].sizeimage = 1280 * 720 * 3 / 2
	spix.plane_fmt[0].bytesperline = 1280
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), VIDIOC_S_FMT, uintptr(unsafe.Pointer(&setFmt)))
	fmt.Printf("S_FMT (1280x720) errno: %v\n", errno)
	if errno == 0 {
		fmt.Printf("S_FMT accepted: %dx%d, planes=%d, size=%d\n",
			spix.width, spix.height, spix.num_planes, spix.plane_fmt[0].sizeimage)
	}
}
