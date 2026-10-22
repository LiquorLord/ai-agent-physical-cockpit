package main

import (
	"fmt"
	"log"
	"net"
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

func triggerVisualFeedback(action string) {
	conn, err := net.Dial("udp", "127.0.0.1:8889")
	if err == nil {
		defer conn.Close()
		conn.Write([]byte(action))
	}
}

func sendGestureKey(action string) {
	log.Printf("[gesture_detector] >>> TRIGGER ACTION: %s <<<", action)

	// Trigger high-contrast screen feedback on touch_cockpit (full-screen border glow + button highlight)
	triggerVisualFeedback(action)

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

func isSkin(y, u, v byte) bool {
	return v >= 133 && v <= 175 && u >= 80 && u <= 130 && y >= 40
}

// Lightweight hand gesture classifier based on YCbCr skin segmentation & morphology
func classifyHandPose(skinGrid []bool, w, h int) string {
	cropX1 := w / 5
	cropX2 := 4 * w / 5
	cropY1 := h / 5
	cropY2 := 4 * h / 5

	skinCount := 0
	minX, maxX := 999, -1
	minY, maxY := 999, -1
	fgColumns := make([]int, cropX2-cropX1)

	for y := cropY1; y < cropY2; y++ {
		row := y * w
		for x := cropX1; x < cropX2; x++ {
			if skinGrid[row+x] {
				skinCount++
				fgColumns[x-cropX1]++
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	// Hand must occupy significant foreground area (at least 380 pixels in 160x90 downsampled grid)
	if skinCount < 380 {
		return ""
	}

	handW := maxX - minX + 1
	handH := maxY - minY + 1
	aspectRatio := float64(handW) / float64(handH)

	// Smooth column histogram with 3-tap moving average
	smoothed := make([]int, len(fgColumns))
	for i := 1; i < len(fgColumns)-1; i++ {
		smoothed[i] = (fgColumns[i-1] + fgColumns[i] + fgColumns[i+1]) / 3
	}

	// Count finger peaks (local maxima above vertical threshold)
	peakThresh := (cropY2 - cropY1) / 4 // at least 1/4 vertical extent
	peaks := 0
	inPeak := false

	for i := 2; i < len(smoothed)-2; i++ {
		val := smoothed[i]
		if val > peakThresh {
			if !inPeak && val > smoothed[i-1] && val >= smoothed[i+1] {
				peaks++
				inPeak = true
			}
		} else {
			inPeak = false
		}
	}

	// 1. Closed Fist ✊ (STOP / ABORT):
	// Compact blob, no finger peaks (peaks <= 1), roughly square/round (0.65 <= W/H <= 1.35)
	if (peaks == 0 || peaks == 1) && aspectRatio >= 0.65 && aspectRatio <= 1.35 && handW >= 16 && handH >= 16 {
		return "STOP_SIGN"
	}

	// 2. Affirmative Gestures (PROCEED / READY):
	// V-Sign (2 peaks) or OK-Sign (3 peaks)
	if peaks == 2 {
		return "V_SIGN"
	} else if peaks == 3 {
		return "OK_SIGN"
	} else if peaks >= 4 {
		return "STOP_SIGN" // Fallback: also accept 5-finger spread hand
	}

	return ""
}

func main() {
	devPath := "/dev/video8"
	if len(os.Args) > 1 {
		devPath = os.Args[1]
	}

	capWidth := 1920
	capHeight := 1080
	dsStep := 12 // 1920 / 12 = 160, 1080 / 12 = 90

	if devPath == "/dev/video0" {
		capWidth = 1280
		capHeight = 720
		dsStep = 8 // 1280 / 8 = 160, 720 / 8 = 90
	}

	log.Printf("[gesture_detector] Starting AGY Visual Gesture Engine on %s (%dx%d)...", devPath, capWidth, capHeight)

	var fd int
	var err error

	// Wait up to 10s for camera device
	for i := 0; i < 5; i++ {
		fd, err = syscall.Open(devPath, syscall.O_RDWR, 0666)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("[!] Failed to open %s: %v", devPath, err)
	}
	defer syscall.Close(fd)

	log.Printf("[+] Successfully opened %s", devPath)

	// 1. Select Input 0
	var input int32 = 0
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0xc0045627, uintptr(unsafe.Pointer(&input)))

	// 2. Set Video Format (NV12 MPLANE)
	var sfmt v4l2_format_mp
	sfmt.buf_type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	smp := (*v4l2_pix_format_mplane)(unsafe.Pointer(&sfmt.fmt[0]))
	smp.width = uint32(capWidth)
	smp.height = uint32(capHeight)
	smp.pixelformat = V4L2_PIX_FMT_NV12
	smp.field = 1 // V4L2_FIELD_NONE
	smp.num_planes = 1
	smp.plane_fmt[0].sizeimage = uint32(capWidth * capHeight * 3 / 2)
	smp.plane_fmt[0].bytesperline = uint32(capWidth)

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

	log.Printf("[+] User-facing camera stream active on %s! Monitoring for ✌️ V / 👌 OK / ✋ Stop gestures...", devPath)

	var (
		consecutiveReady = 0
		consecutiveStp   = 0
		lastTrigger      time.Time
	)

	frameIndex := 0
	subW := capWidth / dsStep  // 160
	subH := capHeight / dsStep // 90
	subSkin := make([]bool, subW*subH)

	alignedHeight := int(planes[0].length) * 2 / (capWidth * 3)
	if alignedHeight < capHeight {
		alignedHeight = capHeight
	}
	yPlaneSize := capWidth * alignedHeight

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

		frameData := buffers[dqbuf.index].start[:buffers[dqbuf.index].length]
		frameIndex++

		// Process every 2nd frame (~12.5 fps processing rate) for snappy responsiveness
		if frameIndex%2 == 0 {
			// Fast spatial downsampling with YCbCr skin detection
			idx := 0
			for y := 0; y < capHeight; y += dsStep {
				rowY := y * capWidth
				uvRow := yPlaneSize + (y/2)*capWidth
				for x := 0; x < capWidth; x += dsStep {
					if idx < len(subSkin) {
						yVal := frameData[rowY+x]
						uvIdx := uvRow + (x/2)*2
						var uVal, vVal byte = 128, 128
						if uvIdx+1 < len(frameData) {
							uVal = frameData[uvIdx]
							vVal = frameData[uvIdx+1]
						}
						subSkin[idx] = isSkin(yVal, uVal, vVal)
						idx++
					}
				}
			}

			pose := classifyHandPose(subSkin, subW, subH)

			if pose == "V_SIGN" || pose == "OK_SIGN" {
				consecutiveReady++
				consecutiveStp = 0
				log.Printf("[*] Detected potential confirm gesture (%s, streak=%d)...", pose, consecutiveReady)
			} else if pose == "STOP_SIGN" {
				consecutiveStp++
				consecutiveReady = 0
				log.Printf("[*] Detected potential STOP gesture (streak=%d)...", consecutiveStp)
			} else {
				if consecutiveReady > 0 {
					consecutiveReady--
				}
				if consecutiveStp > 0 {
					consecutiveStp--
				}
			}

			// Temporal debounce: 2 consecutive frames, 1.5s cooldown
			if time.Since(lastTrigger) > 1500*time.Millisecond {
				if consecutiveReady >= 2 {
					lastTrigger = time.Now()
					consecutiveReady = 0
					consecutiveStp = 0
					log.Printf("[+] GESTURE CONFIRMED: ✌️/👌 detected! Emitting 'ready'...")
					go sendGestureKey("ready")
				} else if consecutiveStp >= 2 {
					lastTrigger = time.Now()
					consecutiveReady = 0
					consecutiveStp = 0
					log.Printf("[+] GESTURE CONFIRMED: ✋ STOP detected! Emitting 'ctrl+c'...")
					go sendGestureKey("ctrl+c")
				}
			}
		}

		// Re-queue buffer
		syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_QBUF, uintptr(unsafe.Pointer(&dqbuf)))
	}
}
