package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// Linux input_event definition on 32-bit ARM (16 bytes)
type InputEvent struct {
	TimeSec  int32
	TimeUsec int32
	Type     uint16
	Code     uint16
	Value    int32
}

const (
	EV_SYN             = 0x00
	EV_KEY             = 0x01
	EV_ABS             = 0x03
	SYN_REPORT         = 0x00
	SYN_MT_REPORT      = 0x02
	BTN_TOUCH          = 0x14a // 330
	ABS_X              = 0x00
	ABS_Y              = 0x01
	ABS_MT_SLOT        = 0x2f // 47
	ABS_MT_TOUCH_MAJOR = 0x30 // 48
	ABS_MT_POSITION_X  = 0x35 // 53
	ABS_MT_POSITION_Y  = 0x36 // 54
	ABS_MT_TRACKING_ID = 0x39 // 57
	ABS_MT_PRESSURE    = 0x3a // 58

	ScreenWidth  = 1600
	ScreenHeight = 600

	// RGB565 Colors
	ColorBlack      = uint16(0x0000)
	ColorWhite      = uint16(0xFFFF)
	ColorGreen      = uint16(0x07E0) // Emerald Green
	ColorGreenDark  = uint16(0x03E0)
	ColorGreenLight = uint16(0x97F2)
	ColorRed        = uint16(0xF800) // Crimson Red
	ColorRedDark    = uint16(0x8000)
	ColorRedLight   = uint16(0xFCB2)
	ColorGray       = uint16(0x4208)
	ColorDarkGray   = uint16(0x2104)

	FBIOGET_VSCREENINFO = 0x4600
	FBIOPAN_DISPLAY     = 0x4606
	FB_ACTIVATE_NOW     = 0
	FB_ACTIVATE_FORCE   = 128
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

type CockpitUI struct {
	mu     sync.Mutex
	fbFile *os.File
	fbMem  []uint16
	vinfo  FbVarScreeninfo
}

func NewCockpitUI() (*CockpitUI, error) {
	fb, err := os.OpenFile("/dev/fb0", os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("open /dev/fb0 failed: %v", err)
	}

	fbBytes, err := syscall.Mmap(int(fb.Fd()), 0, ScreenWidth*ScreenHeight*2, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		fb.Close()
		return nil, fmt.Errorf("mmap /dev/fb0 failed: %v", err)
	}

	ui := &CockpitUI{
		fbFile: fb,
		fbMem:  unsafe.Slice((*uint16)(unsafe.Pointer(&fbBytes[0])), ScreenWidth*ScreenHeight),
	}

	// Query screeninfo
	syscall.Syscall(syscall.SYS_IOCTL, fb.Fd(), FBIOGET_VSCREENINFO, uintptr(unsafe.Pointer(&ui.vinfo)))
	ui.vinfo.Activate = FB_ACTIVATE_NOW | FB_ACTIVATE_FORCE
	ui.vinfo.Xoffset = 0
	ui.vinfo.Yoffset = 0

	return ui, nil
}

func (ui *CockpitUI) DrawRect(x, y, w, h int, color uint16) {
	for r := y; r < y+h && r < ScreenHeight; r++ {
		if r < 0 {
			continue
		}
		rowStart := r * ScreenWidth
		for c := x; c < x+w && c < ScreenWidth; c++ {
			if c < 0 {
				continue
			}
			ui.fbMem[rowStart+c] = color
		}
	}
}

func (ui *CockpitUI) DrawFrame(x, y, w, h, border int, color uint16) {
	ui.DrawRect(x, y, w, border, color)                 // Top
	ui.DrawRect(x, y+h-border, w, border, color)        // Bottom
	ui.DrawRect(x, y, border, h, color)                 // Left
	ui.DrawRect(x+w-border, y, border, h, color)        // Right
}

// 5x7 Basic Bitmap Font for Button Labels
var font5x7 = map[rune][]byte{
	'R': {0x7E, 0x11, 0x31, 0x5E, 0x48, 0x44, 0x43},
	'E': {0x7F, 0x49, 0x49, 0x49, 0x41, 0x41, 0x41},
	'A': {0x3E, 0x49, 0x49, 0x49, 0x3E, 0x00, 0x00},
	'D': {0x7F, 0x41, 0x41, 0x22, 0x1C, 0x00, 0x00},
	'Y': {0x07, 0x08, 0x70, 0x08, 0x07, 0x00, 0x00},
	'P': {0x7F, 0x09, 0x09, 0x09, 0x06, 0x00, 0x00},
	'O': {0x3E, 0x41, 0x41, 0x41, 0x3E, 0x00, 0x00},
	'C': {0x3E, 0x41, 0x41, 0x41, 0x22, 0x00, 0x00},
	'S': {0x46, 0x49, 0x49, 0x49, 0x31, 0x00, 0x00},
	'T': {0x01, 0x01, 0x7F, 0x01, 0x01, 0x00, 0x00},
	'B': {0x7F, 0x49, 0x49, 0x49, 0x36, 0x00, 0x00},
	' ': {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
}

func (ui *CockpitUI) DrawText(x, y int, text string, scale int, color uint16) {
	currX := x
	for _, ch := range text {
		glyph, ok := font5x7[ch]
		if !ok {
			glyph = font5x7[' ']
		}
		for col := 0; col < len(glyph); col++ {
			bits := glyph[col]
			for row := 0; row < 8; row++ {
				if (bits & (1 << row)) != 0 {
					ui.DrawRect(currX+col*scale, y+row*scale, scale, scale, color)
				}
			}
		}
		currX += (len(glyph) + 1) * scale
	}
}

func (ui *CockpitUI) renderButtons(activeButton int) {
	// 1. Right Action Bar Container (X: 1200..1600)
	ui.DrawRect(1200, 0, 400, ScreenHeight, ColorDarkGray)
	ui.DrawRect(1198, 0, 4, ScreenHeight, ColorGray) // Divider border

	// 2. Button 1: [READY / PROCEED] (Y: 50..270)
	btn1Color := ColorGreenDark
	btn1Border := ColorGreen
	if activeButton == 1 {
		btn1Color = ColorWhite
		btn1Border = ColorGreenLight
	}
	ui.DrawRect(1240, 50, 320, 220, btn1Color)
	ui.DrawFrame(1240, 50, 320, 220, 4, btn1Border)
	ui.DrawText(1290, 110, "PROCEED", 4, ColorWhite)
	ui.DrawText(1320, 170, "READY", 3, ColorGreenLight)

	// 3. Button 2: [STOP / ABORT] (Y: 330..550)
	btn2Color := ColorRedDark
	btn2Border := ColorRed
	if activeButton == 2 {
		btn2Color = ColorWhite
		btn2Border = ColorRedLight
	}
	ui.DrawRect(1240, 330, 320, 220, btn2Color)
	ui.DrawFrame(1240, 330, 320, 220, 4, btn2Border)
	ui.DrawText(1320, 390, "STOP", 4, ColorWhite)
	ui.DrawText(1325, 450, "ABORT", 3, ColorRedLight)
}

func (ui *CockpitUI) RenderActionBar(activeButton int) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.renderButtons(activeButton)
	ui.Flip()
}

func (ui *CockpitUI) FlashFeedback(action string) {
	ui.mu.Lock()
	if action == "ready" {
		ui.renderButtons(1)
		ui.Flip()
		ui.mu.Unlock()

		time.Sleep(300 * time.Millisecond)

		ui.mu.Lock()
		ui.renderButtons(0)
		ui.Flip()
		ui.mu.Unlock()
	} else if action == "ctrl+c" {
		ui.renderButtons(2)
		ui.Flip()
		ui.mu.Unlock()

		time.Sleep(300 * time.Millisecond)

		ui.mu.Lock()
		ui.renderButtons(0)
		ui.Flip()
		ui.mu.Unlock()
	} else {
		ui.mu.Unlock()
	}
}

func (ui *CockpitUI) Flip() {
	syscall.Syscall(syscall.SYS_IOCTL, ui.fbFile.Fd(), FBIOPAN_DISPLAY, uintptr(unsafe.Pointer(&ui.vinfo)))
}

func sendKeystroke(action string) {
	log.Printf("[touch_cockpit] Triggering Action: %s", action)

	// Append to /tmp/cockpit_event.log for host companion monitoring
	if f, err := os.OpenFile("/tmp/cockpit_event.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
		fmt.Fprintf(f, "%s\n", action)
		f.Close()
	}

	// Direct attempt via /dev/hidg0
	hid, err := os.OpenFile("/dev/hidg0", os.O_WRONLY, 0666)
	if err == nil {
		defer hid.Close()
		switch action {
		case "ready":
			// Injects "ready\n"
			keys := []byte{
				0x15, // r
				0x08, // e
				0x04, // a
				0x07, // d
				0x1c, // y
				0x28, // enter
			}
			for _, k := range keys {
				press := []byte{0, 0, k, 0, 0, 0, 0, 0}
				release := []byte{0, 0, 0, 0, 0, 0, 0, 0}
				hid.Write(press)
				time.Sleep(15 * time.Millisecond)
				hid.Write(release)
				time.Sleep(15 * time.Millisecond)
			}
			return
		case "ctrl+c":
			press := []byte{0x01, 0, 0x06, 0, 0, 0, 0, 0} // LeftCtrl + c
			release := []byte{0, 0, 0, 0, 0, 0, 0, 0}
			hid.Write(press)
			time.Sleep(20 * time.Millisecond)
			hid.Write(release)
			return
		}
	}

	// Fallback to /mnt/sdcard/send_hid_key executable
	cmd := exec.Command("/mnt/sdcard/send_hid_key", action)
	if err := cmd.Run(); err != nil {
		log.Printf("[-] send_hid_key execution failed: %v", err)
	}
}

func main() {
	log.Println("[touch_cockpit] Initializing AGY Touch Controller on /dev/input/event0...")

	ui, err := NewCockpitUI()
	if err != nil {
		log.Printf("[!] UI init warning: %v (continuing touch event loop)", err)
	} else {
		ui.RenderActionBar(0)
		log.Println("[+] Action Bar rendered on 1600x600 screen")
	}

	// Start inter-process visual feedback listener (from gesture_detector)
	go func() {
		addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:8889")
		if err != nil {
			log.Printf("[-] UDP resolve error: %v", err)
			return
		}
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			log.Printf("[-] UDP listen error: %v", err)
			return
		}
		defer conn.Close()
		log.Println("[+] Inter-process visual feedback listener active on UDP :8889")
		recvBuf := make([]byte, 64)
		for {
			n, _, err := conn.ReadFrom(recvBuf)
			if err != nil {
				continue
			}
			action := strings.TrimSpace(string(recvBuf[:n]))
			log.Printf("[touch_cockpit] Received gesture visual feedback event: %s", action)
			if ui != nil {
				go ui.FlashFeedback(action)
			}
		}
	}()

	evFile, err := os.Open("/dev/input/event0")
	if err != nil {
		log.Fatalf("[!] Failed to open /dev/input/event0: %v", err)
	}
	defer evFile.Close()

	var (
		rawX       int32 = 0
		rawY       int32 = 0
		pressure   int32 = 1 // default positive until explicit 0
		hasCoord         = false
		lastAction time.Time
	)

	buf := make([]byte, 16)
	for {
		_, err := io.ReadFull(evFile, buf)
		if err != nil {
			log.Fatalf("[-] Error reading event0: %v", err)
		}

		var ev InputEvent
		ev.TimeSec = int32(binary.LittleEndian.Uint32(buf[0:4]))
		ev.TimeUsec = int32(binary.LittleEndian.Uint32(buf[4:8]))
		ev.Type = binary.LittleEndian.Uint16(buf[8:10])
		ev.Code = binary.LittleEndian.Uint16(buf[10:12])
		ev.Value = int32(binary.LittleEndian.Uint32(buf[12:16]))

		// Log raw events for 100% observability
		log.Printf("[RAW_EV] Type=0x%02x Code=0x%02x Val=%d", ev.Type, ev.Code, ev.Value)

		if ev.Type == EV_ABS {
			switch ev.Code {
			case ABS_MT_POSITION_X, ABS_X:
				rawX = ev.Value
				hasCoord = true
			case ABS_MT_POSITION_Y, ABS_Y:
				rawY = ev.Value
				hasCoord = true
			case ABS_MT_PRESSURE:
				pressure = ev.Value
				if pressure > 0 {
					hasCoord = true
				}
			case ABS_MT_TRACKING_ID:
				if ev.Value == -1 {
					pressure = 0
					hasCoord = false
				} else {
					pressure = 1
				}
			}
		} else if ev.Type == EV_KEY && ev.Code == BTN_TOUCH {
			if ev.Value == 1 {
				pressure = 1
			} else {
				pressure = 0
				hasCoord = false
			}
		} else if ev.Type == EV_SYN {
			if ev.Code == SYN_REPORT || ev.Code == SYN_MT_REPORT {
				if hasCoord && pressure > 0 {
					// 1:1 Direct mapping (hardware GT911 firmware outputs in native 1600x600 landscape)
					screenX := int(rawX)
					screenY := int(rawY)

					log.Printf("[*] Touch detected: Raw(%d, %d) -> Screen(%d, %d)", rawX, rawY, screenX, screenY)

					// Debounce triggers by at least 600ms
					if time.Since(lastAction) >= 600*time.Millisecond {
						// Button 1: [READY / PROCEED]
						if screenX >= 1200 && screenY >= 50 && screenY <= 290 {
							lastAction = time.Now()
							log.Printf("[HIT] Button 1 [READY / PROCEED] Hit at Screen(%d, %d)!", screenX, screenY)
							if ui != nil {
								ui.RenderActionBar(1) // Flash button 1
							}
							go sendKeystroke("ready")
							go func() {
								time.Sleep(200 * time.Millisecond)
								if ui != nil {
									ui.RenderActionBar(0)
								}
							}()
						} else if screenX >= 1200 && screenY >= 310 && screenY <= 560 {
							// Button 2: [STOP / ABORT]
							lastAction = time.Now()
							log.Printf("[HIT] Button 2 [STOP / ABORT] Hit at Screen(%d, %d)!", screenX, screenY)
							if ui != nil {
								ui.RenderActionBar(2) // Flash button 2
							}
							go sendKeystroke("ctrl+c")
							go func() {
								time.Sleep(200 * time.Millisecond)
								if ui != nil {
									ui.RenderActionBar(0)
								}
							}()
						}
					}
				}
				if pressure == 0 {
					hasCoord = false
				}
			}
		}
	}
}

