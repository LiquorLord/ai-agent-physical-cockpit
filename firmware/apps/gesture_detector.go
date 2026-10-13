package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

// Standard V4L2 ioctl constants
const (
	VIDIOC_QUERYCAP  = 0x80685600
	VIDIOC_S_FMT     = 0xc0d05605 // 32-bit v4l2_format
	VIDIOC_REQBUFS   = 0xc0145608
	VIDIOC_QUERYBUF  = 0xc0445609
	VIDIOC_QBUF      = 0xc044560f
	VIDIOC_DQBUF     = 0xc0445611
	VIDIOC_STREAMON  = 0x40045612
	VIDIOC_STREAMOFF = 0x40045613

	V4L2_BUF_TYPE_VIDEO_CAPTURE = 1
	V4L2_MEMORY_MMAP            = 1

	V4L2_PIX_FMT_NV12 = 0x3231564E // 'NV12'
	V4L2_PIX_FMT_YUYV = 0x56595559 // 'YUYV'

	CapWidth  = 640
	CapHeight = 480

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

type v4l2_timeval struct {
	tv_sec  int32
	tv_usec int32
}

type v4l2_buffer struct {
	index     uint32
	buf_type  uint32
	bytesused uint32
	flags     uint32
	field     uint32
	timestamp v4l2_timeval
	timecode  [16]byte
	sequence  uint32
	memory    uint32
	offset    uint32 // m.offset
	length    uint32
	reserved2 uint32
	reserved  uint32
}

type v4l2_pix_format struct {
	width        uint32
	height       uint32
	pixelformat  uint32
	field        uint32
	bytesperline uint32
	sizeimage    uint32
	colorspace   uint32
	priv         uint32
	flags        uint32
	ycbcr_enc    uint32
	quantization uint32
	xfer_func    uint32
}

type v4l2_format struct {
	buf_type uint32
	fmt      [200]byte // union encompassing v4l2_pix_format
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

	// Direct HID injection
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
	// Sample center window (320x240 inside 640x480)
	cropX1 := w / 4
	cropX2 := 3 * w / 4
	cropY1 := h / 4
	cropY2 := 3 * h / 4

	// 1. Calculate average luminance in crop window
	var sum int64
	count := 0
	for y := cropY1; y < cropY2; y += 4 {
		row := y * w
		for x := cropX1; x < cropX2; x += 4 {
			sum += int64(gray[row+x])
			count++
		}
	}
	if count == 0 {
		return ""
	}
	avgLum := byte(sum / int64(count))

	// If too dark or too bright, skip
	if avgLum < 30 || avgLum > 230 {
		return ""
	}

	// 2. Count columns with vertical foreground extent (finger columns)
	// Fingertips appear as distinct column peaks separated by background valleys
	fgColumns := make([]int, (cropX2-cropX1)/4)
	colIdx := 0
	for x := cropX1; x < cropX2; x += 4 {
		fgPixels := 0
		for y := cropY1; y < cropY2; y += 4 {
			val := gray[y*w+x]
			// Difference from local average indicates foreground hand
			diff := int(val) - int(avgLum)
			if diff > 25 || diff < -25 {
				fgPixels++
			}
		}
		fgColumns[colIdx] = fgPixels
		colIdx++
	}

	// 3. Count peaks in column profile
	peaks := 0
	inPeak := false
	peakWidth := 0
	for _, val := range fgColumns {
		if val > 12 { // Threshold for hand presence in column
			if !inPeak {
				inPeak = true
				peakWidth = 1
			} else {
				peakWidth++
			}
		} else {
			if inPeak {
				if peakWidth >= 2 && peakWidth <= 12 {
					peaks++
				}
				inPeak = false
				peakWidth = 0
			}
		}
	}
	if inPeak && peakWidth >= 2 && peakWidth <= 12 {
		peaks++
	}

	// Classify:
	// 2 distinct peaks -> V-Sign / Peace
	// 3 distinct peaks + compact palm -> OK Gesture
	// 5 distinct peaks -> Stop
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

	// Wait up to 30s for camera device to become available
	for i := 0; i < 15; i++ {
		fd, err = syscall.Open("/dev/video0", syscall.O_RDWR|syscall.O_NONBLOCK, 0666)
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("[!] Failed to open /dev/video0: %v", err)
	}
	defer syscall.Close(fd)

	log.Println("[+] Successfully opened /dev/video0")

	// Set Video Format (NV12 or YUYV 640x480)
	var fmtReq v4l2_format
	fmtReq.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE
	pix := (*v4l2_pix_format)(unsafe.Pointer(&fmtReq.fmt[0]))
	pix.width = CapWidth
	pix.height = CapHeight
	pix.pixelformat = V4L2_PIX_FMT_NV12
	pix.field = 1 // V4L2_FIELD_NONE

	syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_S_FMT, uintptr(unsafe.Pointer(&fmtReq)))

	// Request 2 MMAP buffers
	var req v4l2_requestbuffers
	req.count = 2
	req.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE
	req.memory = V4L2_MEMORY_MMAP
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_REQBUFS, uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		log.Fatalf("[-] VIDIOC_REQBUFS failed: %v", errno)
	}

	// Map buffers
	type Buffer struct {
		start  []byte
		length int
	}
	buffers := make([]Buffer, req.count)

	for i := uint32(0); i < req.count; i++ {
		var buf v4l2_buffer
		buf.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE
		buf.memory = V4L2_MEMORY_MMAP
		buf.index = i

		syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_QUERYBUF, uintptr(unsafe.Pointer(&buf)))

		mmap, err := syscall.Mmap(fd, int64(buf.offset), int(buf.length), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
		if err != nil {
			log.Fatalf("[-] Mmap failed for buf %d: %v", i, err)
		}
		buffers[i] = Buffer{start: mmap, length: int(buf.length)}

		// Queue buffer
		syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_QBUF, uintptr(unsafe.Pointer(&buf)))
	}

	// Start video streaming
	streamType := uint32(V4L2_BUF_TYPE_VIDEO_CAPTURE)
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_STREAMON, uintptr(unsafe.Pointer(&streamType)))
	defer syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_STREAMOFF, uintptr(unsafe.Pointer(&streamType)))

	log.Println("[+] Camera stream active! Monitoring for ✌️ V / 👌 OK / ✋ Stop gestures...")

	var (
		consecutiveV   = 0
		consecutiveOK  = 0
		consecutiveStp = 0
		lastTrigger    time.Time
	)

	frameIndex := 0
	for {
		var buf v4l2_buffer
		buf.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE
		buf.memory = V4L2_MEMORY_MMAP

		// Dequeue frame
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_DQBUF, uintptr(unsafe.Pointer(&buf)))
		if errno != 0 {
			if errno == syscall.EAGAIN {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			time.Sleep(20 * time.Millisecond)
			continue
		}

		frameData := buffers[buf.index].start[:CapWidth*CapHeight] // First W*H bytes = Grayscale plane in NV12
		frameIndex++

		// Process every 2nd frame to save CPU (~15 fps processing rate)
		if frameIndex%2 == 0 {
			pose := classifyHandPose(frameData, CapWidth, CapHeight)

			if pose == "V_SIGN" {
				consecutiveV++
				consecutiveOK = 0
				consecutiveStp = 0
			} else if pose == "OK_SIGN" {
				consecutiveOK++
				consecutiveV = 0
				consecutiveStp = 0
			} else if pose == "STOP_SIGN" {
				consecutiveStp++
				consecutiveV = 0
				consecutiveOK = 0
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
					log.Printf("[+] GESTURE CONFIRMED: V / OK detected! Sending 'ready\\n' to host...")
					go flashFeedbackBorder(0x07E0) // Green border
					go sendGestureKey("ready")
				} else if consecutiveStp >= 3 {
					lastTrigger = time.Now()
					consecutiveStp = 0
					log.Printf("[+] GESTURE CONFIRMED: STOP detected! Sending 'Ctrl+C' to host...")
					go flashFeedbackBorder(0xF800) // Red border
					go sendGestureKey("ctrl+c")
				}
			}
		}

		// Re-queue buffer
		syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_QBUF, uintptr(unsafe.Pointer(&buf)))
	}
}
