package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
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
	ColorCyan       = uint16(0x07FF)
	ColorYellow     = uint16(0xFFE0)
	ColorGray       = uint16(0x7BEF)
	ColorDarkGray   = uint16(0x18C3)
	ColorDivider    = uint16(0x3186)

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

type SystemMetrics struct {
	TempC          float32
	CpuUsage       int
	RamUsedMB      int
	RamTotalMB     int
	StreamFps      string
	LastAction     string
	LastActionTime string
}

type CockpitUI struct {
	mu      sync.Mutex
	fbFile  *os.File
	fbMem   []uint16
	vinfo   FbVarScreeninfo
	metrics SystemMetrics
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
		metrics: SystemMetrics{
			StreamFps:  "IDLE",
			LastAction: "READY FOR INPUT",
		},
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
	ui.DrawRect(x, y, w, border, color)          // Top
	ui.DrawRect(x, y+h-border, w, border, color) // Bottom
	ui.DrawRect(x, y, border, h, color)          // Left
	ui.DrawRect(x+w-border, y, border, h, color) // Right
}

// 5x7 ASCII Bitmap Font
var font5x7 = map[rune][]byte{
	' ': {0x00, 0x00, 0x00, 0x00, 0x00},
	'!': {0x00, 0x00, 0x5F, 0x00, 0x00},
	'"': {0x00, 0x07, 0x00, 0x07, 0x00},
	'#': {0x14, 0x7F, 0x14, 0x7F, 0x14},
	'$': {0x24, 0x2A, 0x7F, 0x2A, 0x12},
	'%': {0x23, 0x13, 0x08, 0x64, 0x62},
	'&': {0x36, 0x49, 0x55, 0x22, 0x50},
	'\'': {0x00, 0x05, 0x03, 0x00, 0x00},
	'(': {0x00, 0x1C, 0x22, 0x41, 0x00},
	')': {0x00, 0x41, 0x22, 0x1C, 0x00},
	'*': {0x14, 0x08, 0x3E, 0x08, 0x14},
	'+': {0x08, 0x08, 0x3E, 0x08, 0x08},
	',': {0x00, 0x50, 0x30, 0x00, 0x00},
	'-': {0x08, 0x08, 0x08, 0x08, 0x08},
	'.': {0x00, 0x60, 0x60, 0x00, 0x00},
	'/': {0x20, 0x10, 0x08, 0x04, 0x02},
	'0': {0x3E, 0x51, 0x49, 0x45, 0x3E},
	'1': {0x00, 0x42, 0x7F, 0x40, 0x00},
	'2': {0x42, 0x61, 0x51, 0x49, 0x46},
	'3': {0x21, 0x41, 0x45, 0x4B, 0x31},
	'4': {0x18, 0x14, 0x12, 0x7F, 0x10},
	'5': {0x27, 0x45, 0x45, 0x45, 0x39},
	'6': {0x3C, 0x4A, 0x49, 0x49, 0x30},
	'7': {0x01, 0x71, 0x09, 0x05, 0x03},
	'8': {0x36, 0x49, 0x49, 0x49, 0x36},
	'9': {0x06, 0x49, 0x49, 0x29, 0x1E},
	':': {0x00, 0x36, 0x36, 0x00, 0x00},
	';': {0x00, 0x56, 0x36, 0x00, 0x00},
	'<': {0x08, 0x14, 0x22, 0x41, 0x00},
	'=': {0x14, 0x14, 0x14, 0x14, 0x14},
	'>': {0x00, 0x41, 0x22, 0x14, 0x08},
	'?': {0x02, 0x01, 0x51, 0x09, 0x06},
	'@': {0x32, 0x49, 0x79, 0x41, 0x3E},
	'A': {0x7E, 0x11, 0x11, 0x11, 0x7E},
	'B': {0x7F, 0x49, 0x49, 0x49, 0x36},
	'C': {0x3E, 0x41, 0x41, 0x41, 0x22},
	'D': {0x7F, 0x41, 0x41, 0x22, 0x1C},
	'E': {0x7F, 0x49, 0x49, 0x49, 0x41},
	'F': {0x7F, 0x09, 0x09, 0x09, 0x01},
	'G': {0x3E, 0x41, 0x49, 0x49, 0x7A},
	'H': {0x7F, 0x08, 0x08, 0x08, 0x7F},
	'I': {0x00, 0x41, 0x7F, 0x41, 0x00},
	'J': {0x20, 0x40, 0x41, 0x3F, 0x01},
	'K': {0x7F, 0x08, 0x14, 0x22, 0x41},
	'L': {0x7F, 0x40, 0x40, 0x40, 0x40},
	'M': {0x7F, 0x02, 0x0C, 0x02, 0x7F},
	'N': {0x7F, 0x04, 0x08, 0x10, 0x7F},
	'O': {0x3E, 0x41, 0x41, 0x41, 0x3E},
	'P': {0x7F, 0x09, 0x09, 0x09, 0x06},
	'Q': {0x3E, 0x41, 0x51, 0x21, 0x5E},
	'R': {0x7F, 0x09, 0x19, 0x29, 0x46},
	'S': {0x46, 0x49, 0x49, 0x49, 0x31},
	'T': {0x01, 0x01, 0x7F, 0x01, 0x01},
	'U': {0x3F, 0x40, 0x40, 0x40, 0x3F},
	'V': {0x1F, 0x20, 0x40, 0x20, 0x1F},
	'W': {0x7F, 0x20, 0x18, 0x20, 0x7F},
	'X': {0x63, 0x14, 0x08, 0x14, 0x63},
	'Y': {0x07, 0x08, 0x70, 0x08, 0x07},
	'Z': {0x61, 0x51, 0x49, 0x45, 0x43},
	'[': {0x00, 0x7F, 0x41, 0x41, 0x00},
	'\\': {0x02, 0x04, 0x08, 0x10, 0x20},
	']': {0x00, 0x41, 0x41, 0x7F, 0x00},
	'^': {0x04, 0x02, 0x01, 0x02, 0x04},
	'_': {0x40, 0x40, 0x40, 0x40, 0x40},
	'|': {0x00, 0x00, 0x7F, 0x00, 0x00},
}

func (ui *CockpitUI) DrawText(x, y int, text string, scale int, color uint16) {
	currX := x
	for _, ch := range text {
		chUpper := unicode.ToUpper(ch)
		glyph, ok := font5x7[chUpper]
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
	ui.DrawRect(1198, 0, 3, ScreenHeight, ColorDivider) // Divider border

	// 2. Top Header HUD (Y: 0..48)
	clockStr := time.Now().Format("15:04:05")
	ui.DrawText(1240, 10, clockStr, 2, ColorCyan)
	tempStr := fmt.Sprintf("SOC: %.1fC  CPU: %d%%", ui.metrics.TempC, ui.metrics.CpuUsage)
	ui.DrawText(1240, 32, tempStr, 1, ColorGreenLight)

	// 3. Button 1: [READY / PROCEED] (Y: 55..275)
	btn1Color := ColorGreenDark
	btn1Border := ColorGreen
	if activeButton == 1 {
		btn1Color = ColorWhite
		btn1Border = ColorGreenLight
	}
	ui.DrawRect(1230, 55, 340, 220, btn1Color)
	ui.DrawFrame(1230, 55, 340, 220, 4, btn1Border)
	ui.DrawText(1280, 115, "PROCEED", 4, ColorWhite)
	ui.DrawText(1325, 175, "READY", 3, ColorGreenLight)

	// 4. Middle Telemetry Bar (Y: 285..325)
	streamStr := fmt.Sprintf("STREAM: %s", ui.metrics.StreamFps)
	streamColor := ColorYellow
	if strings.Contains(ui.metrics.StreamFps, "FPS") {
		streamColor = ColorCyan
	}
	ui.DrawText(1240, 290, streamStr, 1, streamColor)
	ramStr := fmt.Sprintf("RAM: %dM / %dM", ui.metrics.RamUsedMB, ui.metrics.RamTotalMB)
	ui.DrawText(1240, 308, ramStr, 1, ColorGray)

	// 5. Button 2: [STOP / ABORT] (Y: 335..555)
	btn2Color := ColorRedDark
	btn2Border := ColorRed
	if activeButton == 2 {
		btn2Color = ColorWhite
		btn2Border = ColorRedLight
	}
	ui.DrawRect(1230, 335, 340, 220, btn2Color)
	ui.DrawFrame(1230, 335, 340, 220, 4, btn2Border)
	ui.DrawText(1330, 395, "STOP", 4, ColorWhite)
	ui.DrawText(1325, 455, "ABORT", 3, ColorRedLight)

	// 6. Bottom Footer (Y: 565..595)
	ui.DrawText(1240, 568, "AGY COCKPIT • V553 CORTEX-A7", 1, ColorDivider)
	lastStr := ui.metrics.LastAction
	if ui.metrics.LastActionTime != "" {
		lastStr = fmt.Sprintf("LAST: %s (%s)", ui.metrics.LastAction, ui.metrics.LastActionTime)
	}
	ui.DrawText(1240, 582, lastStr, 1, ColorGreenLight)
}

func (ui *CockpitUI) RenderActionBar(activeButton int) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.renderButtons(activeButton)
	ui.Flip()
}

func (ui *CockpitUI) FlashFeedback(action string) {
	ui.mu.Lock()
	ui.metrics.LastAction = strings.ToUpper(action)
	ui.metrics.LastActionTime = time.Now().Format("15:04:05")
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

func (ui *CockpitUI) StartTelemetryCollector() {
	go func() {
		var prevUser, prevNice, prevSys, prevIdle uint64
		for {
			time.Sleep(1 * time.Second)

			var tempC float32
			if data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp"); err == nil {
				var raw int
				if _, err := fmt.Sscanf(string(data), "%d", &raw); err == nil {
					tempC = float32(raw) / 1000.0
				}
			}

			streamFps := "IDLE"
			if data, err := os.ReadFile("/tmp/stream_fps"); err == nil {
				streamFps = strings.TrimSpace(string(data))
			}

			var ramTotal, ramUsed int
			if data, err := os.ReadFile("/proc/meminfo"); err == nil {
				var total, free, avail int
				for _, line := range strings.Split(string(data), "\n") {
					if strings.HasPrefix(line, "MemTotal:") {
						fmt.Sscanf(line, "MemTotal: %d", &total)
					} else if strings.HasPrefix(line, "MemAvailable:") {
						fmt.Sscanf(line, "MemAvailable: %d", &avail)
					} else if strings.HasPrefix(line, "MemFree:") {
						fmt.Sscanf(line, "MemFree: %d", &free)
					}
				}
				if avail == 0 {
					avail = free
				}
				ramTotal = total / 1024
				ramUsed = (total - avail) / 1024
			}

			cpuUsage := 0
			if data, err := os.ReadFile("/proc/stat"); err == nil {
				var user, nice, sys, idle uint64
				if _, err := fmt.Sscanf(string(data), "cpu %d %d %d %d", &user, &nice, &sys, &idle); err == nil {
					totalDelta := (user + nice + sys + idle) - (prevUser + prevNice + prevSys + prevIdle)
					workDelta := (user + nice + sys) - (prevUser + prevNice + prevSys)
					if totalDelta > 0 {
						cpuUsage = int((workDelta * 100) / totalDelta)
					}
					prevUser, prevNice, prevSys, prevIdle = user, nice, sys, idle
				}
			}

			ui.mu.Lock()
			ui.metrics.TempC = tempC
			ui.metrics.CpuUsage = cpuUsage
			ui.metrics.RamTotalMB = ramTotal
			ui.metrics.RamUsedMB = ramUsed
			ui.metrics.StreamFps = streamFps
			ui.renderButtons(0)
			ui.Flip()
			ui.mu.Unlock()
		}
	}()
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
			keys := []byte{0x15, 0x08, 0x04, 0x07, 0x1c, 0x28} // ready\n
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
	signal.Ignore(syscall.SIGHUP)
	log.Println("[touch_cockpit] Initializing AGY Touch Controller on /dev/input/event0...")

	ui, err := NewCockpitUI()
	if err != nil {
		log.Printf("[!] UI init warning: %v (continuing touch event loop)", err)
	} else {
		ui.RenderActionBar(0)
		ui.StartTelemetryCollector()
		log.Println("[+] Action Bar HUD & Telemetry Collector active")
	}

	// Start inter-process visual feedback listener (from external scripts/gesture)
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
		recvBuf := make([]byte, 64)
		for {
			n, _, err := conn.ReadFrom(recvBuf)
			if err != nil {
				continue
			}
			action := strings.TrimSpace(string(recvBuf[:n]))
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
		pressure   int32 = 1
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

		if ev.Type == EV_ABS {
			switch ev.Code {
			case ABS_MT_POSITION_X, ABS_X:
				rawX = ev.Value
				hasCoord = true
				pressure = 1
			case ABS_MT_POSITION_Y, ABS_Y:
				rawY = ev.Value
				hasCoord = true
				pressure = 1
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
					screenX := int(rawX)
					screenY := int(rawY)

					log.Printf("[TOUCH] Detected at Screen(%d, %d)", screenX, screenY)

					if time.Since(lastAction) >= 400*time.Millisecond {
						// Button 1: [READY / PROCEED] (Y: 40..285)
						if screenX >= 1200 && screenY >= 40 && screenY <= 285 {
							lastAction = time.Now()
							log.Printf("[HIT] Button 1 [READY / PROCEED] Hit at (%d, %d)", screenX, screenY)
							if ui != nil {
								ui.FlashFeedback("ready")
							}
							go sendKeystroke("ready")
						} else if screenX >= 1200 && screenY >= 325 && screenY <= 565 {
							// Button 2: [STOP / ABORT] (Y: 325..565)
							lastAction = time.Now()
							log.Printf("[HIT] Button 2 [STOP / ABORT] Hit at (%d, %d)", screenX, screenY)
							if ui != nil {
								ui.FlashFeedback("ctrl+c")
							}
							go sendKeystroke("ctrl+c")
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
