package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

// Linux Multi-Planar V4L2 ioctl constants
const (
	VIDIOC_QUERYCAP  = 0x80685600
	VIDIOC_G_FMT     = 0xc0cc5604
	VIDIOC_S_FMT     = 0xc0cc5605
	VIDIOC_REQBUFS   = 0xc0145608
	VIDIOC_QUERYBUF  = 0xc0445609
	VIDIOC_QBUF      = 0xc044560f
	VIDIOC_DQBUF     = 0xc0445611
	VIDIOC_STREAMON  = 0x40045612
	VIDIOC_STREAMOFF = 0x40045613

	V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE = 9
	V4L2_MEMORY_MMAP                   = 1

	V4L2_PIX_FMT_NV12 = 0x3231564E // 'NV12'

	CapWidth  = 1280
	CapHeight = 720

	ScreenWidth  = 1600
	ScreenHeight = 600

	FBIOGET_VSCREENINFO = 0x4600
	FBIOPAN_DISPLAY     = 0x4606
	FB_ACTIVATE_NOW     = 0
	FB_ACTIVATE_FORCE   = 128
)

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
	planes    uintptr // *v4l2_plane
	length    uint32  // number of planes = 1
	reserved2 uint32
	reserved  uint32
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

func flashFeedbackBorder(colorRGB565 uint16) {
	fb, err := os.OpenFile("/dev/fb0", os.O_RDWR, 0666)
	if err != nil {
		return
	}
	defer fb.Close()

	var v FbVarScreeninfo
	syscall.Syscall(syscall.SYS_IOCTL, fb.Fd(), FBIOGET_VSCREENINFO, uintptr(unsafe.Pointer(&v)))
	v.Activate = FB_ACTIVATE_NOW | FB_ACTIVATE_FORCE

	// Draw 10px outer border
	border := 10
	pixelBytes := make([]byte, 2)
	pixelBytes[0] = byte(colorRGB565 & 0xFF)
	pixelBytes[1] = byte((colorRGB565 >> 8) & 0xFF)

	// Top & Bottom strips
	strip := make([]byte, ScreenWidth*border*2)
	for i := 0; i < len(strip); i += 2 {
		strip[i] = pixelBytes[0]
		strip[i+1] = pixelBytes[1]
	}

	fb.Seek(0, io.SeekStart)
	fb.Write(strip)
	fb.Seek(int64((ScreenHeight-border)*ScreenWidth*2), io.SeekStart)
	fb.Write(strip)

	syscall.Syscall(syscall.SYS_IOCTL, fb.Fd(), FBIOPAN_DISPLAY, uintptr(unsafe.Pointer(&v)))
}

func sendGestureKey(action string) {
	log.Printf("[gesture_detector] >>> TRIGGER ACTION: %s <<<", action)

	// Append to /tmp/cockpit_event.log for host companion monitoring
	if f, err := os.OpenFile("/tmp/cockpit_event.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
		fmt.Fprintf(f, "%s\n", action)
		f.Close()
	}

	// Direct HID injection if /dev/hidg0 exists
	hid, err := os.OpenFile("/dev/hidg0", os.O_WRONLY, 0666)
	if err == nil {
		defer hid.Close()
		if action == "ready" {
			keys := []byte{0x15, 0x08, 0x04, 0x07, 0x1c, 0x28} // ready\n
			for _, k := range keys {
				hid.Write([]byte{0, 0, k, 0, 0, 0, 0, 0})
				time.Sleep(15 * time.Millisecond)
				hid.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
				time.Sleep(15 * time.Millisecond)
			}
			return
		} else if action == "ctrl+c" {
			hid.Write([]byte{0x01, 0, 0x06, 0, 0, 0, 0, 0})
			time.Sleep(20 * time.Millisecond)
			hid.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
			return
		}
	}

	// Fallback executable
	exec.Command("/mnt/sdcard/send_hid_key", action).Run()
}

// Lightweight hand gesture classifier based on morphological convexity & finger peaks
func classifyHandPose(gray []byte, w, h int) string {
	cropX1 := w / 4
	cropX2 := 3 * w / 4
	cropY1 := h / 4
	cropY2 := 3 * h / 4

	// 1. Calculate average luminance in crop window
	var sum int64
	count := 0
	for y := cropY1; y < cropY2; y += 2 {
		row := y * w
		for x := cropX1; x < cropX2; x += 2 {
			sum += int64(gray[row+x])
			count++
		}
	}
	if count == 0 {
		return ""
	}
	avgLum := byte(sum / int64(count))

	// Skip if too dark or too washed out
	if avgLum < 20 || avgLum > 235 {
		return ""
	}

	// 2. Count columns with vertical foreground extent (finger columns)
	fgColumns := make([]int, (cropX2-cropX1)/2)
	colIdx := 0
	for x := cropX1; x < cropX2; x += 2 {
		fgPixels := 0
		for y := cropY1; y < cropY2; y += 2 {
			val := gray[y*w+x]
			diff := int(val) - int(avgLum)
			if diff > 20 || diff < -20 {
				fgPixels++
			}
		}
		fgColumns[colIdx] = fgPixels
		colIdx++
	}

	// 3. Count peaks in column profile
	peaks := 0
	inPeak := false
	peakThreshold := (cropY2 - cropY1) / 5

	for i := 1; i < len(fgColumns)-1; i++ {
		val := fgColumns[i]
		if val > peakThreshold {
			if !inPeak && val > fgColumns[i-1] {
				peaks++
				inPeak = true
			}
		} else {
			inPeak = false
		}
	}

	// Heuristic Decision Boundary
	if peaks == 2 {
		return "V_SIGN"
	} else if peaks == 3 {
		return "OK_SIGN"
	} else if peaks >= 4 && peaks <= 6 {
		return "STOP_SIGN"
	}

	return ""
}

func main() {
	log.Println("[gesture_detector] Starting AGY Visual Gesture Engine on /dev/video0...")

	var fd int
	var err error

	// Wait up to 10s for camera device
	for i := 0; i < 5; i++ {
		fd, err = syscall.Open("/dev/video0", syscall.O_RDWR, 0666)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("[!] Failed to open /dev/video0: %v", err)
	}
	defer syscall.Close(fd)

	log.Println("[+] Successfully opened /dev/video0")

	// 1. Select Input 0
	var input int32 = 0
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0xc0045627, uintptr(unsafe.Pointer(&input)))

	// 2. Set Video Format (1280x720 NV12 MPLANE)
	var sfmt v4l2_format_mp
	sfmt.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	smp := (*v4l2_pix_format_mplane)(unsafe.Pointer(&sfmt.fmt[0]))
	smp.width = CapWidth
	smp.height = CapHeight
	smp.pixelformat = V4L2_PIX_FMT_NV12
	smp.field = 1 // V4L2_FIELD_NONE
	smp.num_planes = 1
	smp.plane_fmt[0].sizeimage = CapWidth * CapHeight * 3 / 2
	smp.plane_fmt[0].bytesperline = CapWidth

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_S_FMT, uintptr(unsafe.Pointer(&sfmt)))
	if errno != 0 {
		log.Fatalf("[-] VIDIOC_S_FMT failed: %v", errno)
	}
	log.Printf("[+] Negotiated Video Format: %dx%d NV12 (planes=%d)", smp.width, smp.height, smp.num_planes)

	// 3. Request 3 MMAP buffers
	var req v4l2_requestbuffers
	req.count = 3
	req.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	req.memory = V4L2_MEMORY_MMAP
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_REQBUFS, uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		log.Fatalf("[-] VIDIOC_REQBUFS failed: %v", errno)
	}

	// 4. Map buffers
	type Buffer struct {
		start  []byte
		length int
	}
	buffers := make([]Buffer, req.count)
	planes := make([]v4l2_plane, req.count)

	for i := uint32(0); i < req.count; i++ {
		var qbuf v4l2_buffer_mp
		qbuf.index = i
		qbuf.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
		qbuf.memory = V4L2_MEMORY_MMAP
		qbuf.planes = uintptr(unsafe.Pointer(&planes[i]))
		qbuf.length = 1

		syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_QUERYBUF, uintptr(unsafe.Pointer(&qbuf)))

		mmap, err := syscall.Mmap(fd, int64(planes[i].mem_offset), int(planes[i].length), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
		if err != nil {
			log.Fatalf("[-] Mmap failed for buf %d: %v", i, err)
		}
		buffers[i] = Buffer{start: mmap, length: int(planes[i].length)}

		// Queue buffer
		syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_QBUF, uintptr(unsafe.Pointer(&qbuf)))
	}

	// 5. Start streaming
	streamType := uint32(V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE)
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_STREAMON, uintptr(unsafe.Pointer(&streamType)))
	if errno != 0 {
		log.Fatalf("[-] VIDIOC_STREAMON failed: %v", errno)
	}
	defer syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_STREAMOFF, uintptr(unsafe.Pointer(&streamType)))

	log.Println("[+] Camera stream active! Monitoring for ✌️ V / 👌 OK / ✋ Stop gestures...")

	var (
		consecutiveV   = 0
		consecutiveOK  = 0
		consecutiveStp = 0
		lastTrigger    time.Time
	)

	frameIndex := 0
	subW := CapWidth / 8  // 160
	subH := CapHeight / 8 // 90
	subGray := make([]byte, subW*subH)

	for {
		var dqbuf v4l2_buffer_mp
		var dqplane v4l2_plane
		dqbuf.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
		dqbuf.memory = V4L2_MEMORY_MMAP
		dqbuf.planes = uintptr(unsafe.Pointer(&dqplane))
		dqbuf.length = 1

		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_DQBUF, uintptr(unsafe.Pointer(&dqbuf)))
		if errno != 0 {
			if errno == syscall.EAGAIN {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			time.Sleep(20 * time.Millisecond)
			continue
		}

		frameData := buffers[dqbuf.index].start[:CapWidth*CapHeight]
		frameIndex++

		// Process every 3rd frame (~10 fps processing rate) to conserve CPU
		if frameIndex%3 == 0 {
			// Fast 8x spatial downsampling
			idx := 0
			for y := 0; y < CapHeight; y += 8 {
				row := y * CapWidth
				for x := 0; x < CapWidth; x += 8 {
					subGray[idx] = frameData[row+x]
					idx++
				}
			}

			pose := classifyHandPose(subGray, subW, subH)

			if pose == "V_SIGN" {
				consecutiveV++
				consecutiveOK = 0
				consecutiveStp = 0
				log.Println("[*] Detected potential ✌️ V-Sign...")
			} else if pose == "OK_SIGN" {
				consecutiveOK++
				consecutiveV = 0
				consecutiveStp = 0
				log.Println("[*] Detected potential 👌 OK-Sign...")
			} else if pose == "STOP_SIGN" {
				consecutiveStp++
				consecutiveV = 0
				consecutiveOK = 0
				log.Println("[*] Detected potential ✋ Stop-Sign...")
			} else {
				consecutiveV = 0
				consecutiveOK = 0
				consecutiveStp = 0
			}

			// Temporal debounce: 3 consecutive frames
			if time.Since(lastTrigger) > 2*time.Second {
				if consecutiveV >= 3 || consecutiveOK >= 3 {
					lastTrigger = time.Now()
					consecutiveV = 0
					consecutiveOK = 0
					log.Printf("[+] GESTURE CONFIRMED: ✌️/👌 detected! Emitting 'ready'...")
					go flashFeedbackBorder(0x07E0) // Emerald Green border flash
					go sendGestureKey("ready")
				} else if consecutiveStp >= 3 {
					lastTrigger = time.Now()
					consecutiveStp = 0
					log.Printf("[+] GESTURE CONFIRMED: ✋ STOP detected! Emitting 'ctrl+c'...")
					go flashFeedbackBorder(0xF800) // Crimson Red border flash
					go sendGestureKey("ctrl+c")
				}
			}
		}

		// Re-queue buffer
		syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_QBUF, uintptr(unsafe.Pointer(&dqbuf)))
	}
}
