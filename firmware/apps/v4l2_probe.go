package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	VIDIOC_QUERYCAP = 0x80685600
	VIDIOC_G_FMT    = 0xc0cc5604
	VIDIOC_S_FMT    = 0xc0cc5605
	VIDIOC_REQBUFS  = 0xc0145608
	VIDIOC_QUERYBUF = 0xc0445609

	V4L2_BUF_TYPE_VIDEO_CAPTURE        = 1
	V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE = 9
	V4L2_MEMORY_MMAP                   = 1

	V4L2_CAP_VIDEO_CAPTURE        = 0x00000001
	V4L2_CAP_VIDEO_CAPTURE_MPLANE = 0x00001000
	V4L2_CAP_STREAMING            = 0x04000000
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
	fmt      [200]byte
}


type v4l2_requestbuffers struct {
	count    uint32
	buf_type uint32
	memory   uint32
	reserved [2]uint32
}

type v4l2_plane struct {
	bytesused   uint32
	length      uint32
	mem_offset  uint32
	data_offset uint32
	reserved    [11]uint32
}

type v4l2_buffer_mp struct {
	index     uint32
	buf_type  uint32
	bytesused uint32
	flags     uint32
	field     uint32
	timestamp [8]byte
	timecode  [16]byte
	sequence  uint32
	memory    uint32
	planes    uintptr // struct v4l2_plane *
	length    uint32  // number of planes
	reserved2 uint32
	reserved  uint32
}

func fourccStr(val uint32) string {
	return string([]byte{
		byte(val & 0xFF),
		byte((val >> 8) & 0xFF),
		byte((val >> 16) & 0xFF),
		byte((val >> 24) & 0xFF),
	})
}

func main() {
	devPath := "/dev/video0"
	if len(os.Args) > 1 {
		devPath = os.Args[1]
	}

	fmt.Printf("=== V4L2 Probe Tool for %s ===\n", devPath)
	fd, err := syscall.Open(devPath, syscall.O_RDWR, 0666)
	if err != nil {
		fmt.Printf("[-] Failed to open %s: %v\n", devPath, err)
		return
	}
	defer syscall.Close(fd)

	// 1. Query Caps
	var cap v4l2_capability
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_QUERYCAP, uintptr(unsafe.Pointer(&cap)))
	if errno != 0 {
		fmt.Printf("[-] VIDIOC_QUERYCAP failed: %v\n", errno)
		return
	}
	fmt.Printf("[+] Driver: %s\n", string(cap.driver[:]))
	fmt.Printf("[+] Card: %s\n", string(cap.card[:]))
	fmt.Printf("[+] Bus: %s\n", string(cap.bus_info[:]))
	fmt.Printf("[+] Caps: 0x%08x\n", cap.capabilities)
	fmt.Printf("    - CAP_VIDEO_CAPTURE: %v\n", (cap.capabilities&V4L2_CAP_VIDEO_CAPTURE) != 0)
	fmt.Printf("    - CAP_VIDEO_CAPTURE_MPLANE: %v\n", (cap.capabilities&V4L2_CAP_VIDEO_CAPTURE_MPLANE) != 0)
	fmt.Printf("    - CAP_STREAMING: %v\n", (cap.capabilities&V4L2_CAP_STREAMING) != 0)

	// 1.4 Query Input
	var input int32 = 0
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0x80045626, uintptr(unsafe.Pointer(&input)))
	fmt.Printf("[+] Current Input: %d (errno=%v)\n", input, errno)

	input = 0
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0xc0045627, uintptr(unsafe.Pointer(&input)))
	fmt.Printf("[+] Set Input 0: errno=%v\n", errno)

	// 1.5 Enumerate formats
	fmt.Println("[*] Enumerating supported formats (MPLANE)...")
	type v4l2_fmtdesc struct {
		index       uint32
		buf_type    uint32
		flags       uint32
		description [32]byte
		pixelformat uint32
		reserved    [4]uint32
	}
	for i := uint32(0); i < 20; i++ {
		var desc v4l2_fmtdesc
		desc.index = i
		desc.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0xc0405602, uintptr(unsafe.Pointer(&desc)))
		if errno != 0 {
			break
		}
		fmt.Printf("    [%d] %s: %s (0x%08x)\n", i, fourccStr(desc.pixelformat), string(desc.description[:]), desc.pixelformat)
	}

	// 1.6 Enumerate Frame Sizes for NV12
	fmt.Println("[*] Enumerating Frame Sizes for NV12...")
	type v4l2_frmsize_discrete struct {
		width  uint32
		height uint32
	}
	type v4l2_frmsizeenum struct {
		index        uint32
		pixel_format uint32
		fsize_type   uint32
		discrete     v4l2_frmsize_discrete
		reserved     [6]uint32
	}
	for i := uint32(0); i < 10; i++ {
		var fsize v4l2_frmsizeenum
		fsize.index = i
		fsize.pixel_format = 0x3231564E // NV12
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0xc02c564a, uintptr(unsafe.Pointer(&fsize)))
		if errno != 0 {
			fmt.Printf("    FrameSize enum ended at %d (errno=%v)\n", i, errno)
			break
		}
		fmt.Printf("    FrameSize[%d]: %dx%d (type=%d)\n", i, fsize.discrete.width, fsize.discrete.height, fsize.fsize_type)
	}


	var fmtReq v4l2_format_mp
	fmtReq.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_G_FMT, uintptr(unsafe.Pointer(&fmtReq)))
	if errno != 0 {
		fmt.Printf("[-] VIDIOC_G_FMT (MPLANE) failed: %v\n", errno)
	} else {
		mp := (*v4l2_pix_format_mplane)(unsafe.Pointer(&fmtReq.fmt[0]))
		fmt.Printf("[+] Current MPLANE Format:\n")
		fmt.Printf("    Width: %d, Height: %d\n", mp.width, mp.height)
		fmt.Printf("    PixelFormat: %s (0x%08x)\n", fourccStr(mp.pixelformat), mp.pixelformat)
		fmt.Printf("    Num Planes: %d\n", mp.num_planes)
		for p := 0; p < int(mp.num_planes); p++ {
			fmt.Printf("    Plane %d: sizeimage=%d, bytesperline=%d\n", p, mp.plane_fmt[p].sizeimage, mp.plane_fmt[p].bytesperline)
		}
	}

	// 2.5 Set Format (Type 9)
	fmt.Println("[*] Setting format to 1280x720 NV12...")
	var sfmt v4l2_format_mp
	sfmt.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	smp := (*v4l2_pix_format_mplane)(unsafe.Pointer(&sfmt.fmt[0]))
	smp.width = 1280
	smp.height = 720
	smp.pixelformat = 0x3231564E // 'NV12'
	smp.field = 1 // V4L2_FIELD_NONE
	smp.num_planes = 1
	smp.plane_fmt[0].sizeimage = 1280 * 720 * 3 / 2
	smp.plane_fmt[0].bytesperline = 1280

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_S_FMT, uintptr(unsafe.Pointer(&sfmt)))
	if errno != 0 {
		fmt.Printf("[-] VIDIOC_S_FMT (NV12) failed: %v\n", errno)
	} else {
		fmt.Printf("[+] VIDIOC_S_FMT (NV12) SUCCESS! Negotiated: %dx%d, fmt=%s, planes=%d\n",
			smp.width, smp.height, fourccStr(smp.pixelformat), smp.num_planes)
	}

	// 3. Test REQBUFS (Type 9)
	var req v4l2_requestbuffers
	req.count = 2
	req.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	req.memory = V4L2_MEMORY_MMAP
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_REQBUFS, uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		fmt.Printf("[-] VIDIOC_REQBUFS (MPLANE) failed: %v\n", errno)
		return
	}
	fmt.Printf("[+] VIDIOC_REQBUFS (MPLANE) SUCCESS! count=%d\n", req.count)


	// 4. Test QUERYBUF (Type 9)
	var plane v4l2_plane
	var qbuf v4l2_buffer_mp
	qbuf.index = 0
	qbuf.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	qbuf.memory = V4L2_MEMORY_MMAP
	qbuf.planes = uintptr(unsafe.Pointer(&plane))
	qbuf.length = 1

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_QUERYBUF, uintptr(unsafe.Pointer(&qbuf)))
	if errno != 0 {
		fmt.Printf("[-] VIDIOC_QUERYBUF failed: %v\n", errno)
		return
	}
	fmt.Printf("[+] VIDIOC_QUERYBUF SUCCESS!\n")
	fmt.Printf("    Buffer 0 Plane 0: length=%d, offset=0x%x\n", plane.length, plane.mem_offset)

	// Clean up REQBUFS
	req.count = 0
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_REQBUFS, uintptr(unsafe.Pointer(&req)))
	fmt.Println("[+] Probe completed successfully!")
}
