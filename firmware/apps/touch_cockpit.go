package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
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
	ColorBlack       = uint16(0x0000)
	ColorWhite       = uint16(0xFFFF)
	ColorGreen       = uint16(0x07E0) // Emerald Green
	ColorGreenDark   = uint16(0x03E0)
	ColorGreenLight  = uint16(0x97F2)
	ColorRed         = uint16(0xF800) // Crimson Red
	ColorRedDark     = uint16(0x8000)
	ColorRedLight    = uint16(0xFCB2)
	ColorCyan        = uint16(0x07FF)
	ColorCyanDark    = uint16(0x0270)
	ColorCyanLight   = uint16(0x87FF)
	ColorPurple      = uint16(0xA25F) // Royal Violet / Purple
	ColorPurpleDark  = uint16(0x514E)
	ColorPurpleLight = uint16(0xE3BF)
	ColorYellow      = uint16(0xFFE0) // Amber / Gold
	ColorYellowDark  = uint16(0x8400)
	ColorGray        = uint16(0x7BEF)
	ColorLightGray   = uint16(0xC618)
	ColorDarkGray    = uint16(0x18C3)
	ColorDivider     = uint16(0x3186)

	FBIOGET_VSCREENINFO = 0x4600
	FBIOPAN_DISPLAY     = 0x4606
	FB_ACTIVATE_NOW     = 0
	FB_ACTIVATE_FORCE   = 128

	ViewButtons = 0
	ViewMenu    = 1
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

type ButtonConfig struct {
	Label       string
	SubText     string
	Color       uint16
	BorderColor uint16
	LightColor  uint16
	Action      string
	HidKey      string
}

type AgentProfile struct {
	ID         string
	Name       string
	Badge      string
	BadgeColor uint16
	Buttons    [4]ButtonConfig
}

// JSON serialization structures for SD card customization
type JsonButtonConfig struct {
	Label   string `json:"label"`
	SubText string `json:"subtext"`
	Action  string `json:"action"`
	Color   string `json:"color"`
	Key     string `json:"key"`
}

type JsonAgentProfile struct {
	ID      string             `json:"id"`
	Name    string             `json:"name"`
	Badge   string             `json:"badge"`
	Color   string             `json:"color"`
	Buttons [4]JsonButtonConfig `json:"buttons"`
}

type JsonCockpitConfig struct {
	ActiveProfile string             `json:"active_profile"`
	Profiles      []JsonAgentProfile `json:"profiles"`
}

func mapColorName(c string) (color, border, light uint16) {
	switch strings.ToLower(c) {
	case "green":
		return ColorGreenDark, ColorGreen, ColorGreenLight
	case "cyan", "blue":
		return ColorCyanDark, ColorCyan, ColorCyanLight
	case "purple", "magenta":
		return ColorPurpleDark, ColorPurple, ColorPurpleLight
	case "red":
		return ColorRedDark, ColorRed, ColorRedLight
	case "yellow", "gold", "orange":
		return ColorDarkGray, ColorYellow, ColorWhite
	case "gray", "dark":
		return ColorDarkGray, ColorGray, ColorWhite
	default:
		return ColorGreenDark, ColorGreen, ColorGreenLight
	}
}

func mapBadgeColor(c string) uint16 {
	switch strings.ToLower(c) {
	case "green":
		return ColorGreen
	case "cyan", "blue":
		return ColorCyan
	case "purple", "magenta":
		return ColorPurple
	case "red":
		return ColorRed
	case "yellow", "gold", "orange":
		return ColorYellow
	case "gray":
		return ColorGray
	default:
		return ColorGreen
	}
}

func normalizeHidKey(k string) string {
	k = strings.ReplaceAll(k, "\\n", "\n")
	k = strings.ReplaceAll(k, "\\t", "\t")
	k = strings.ReplaceAll(k, "\\r", "\r")
	k = strings.ReplaceAll(k, "^C", "\x03")
	k = strings.ReplaceAll(k, "^c", "\x03")
	k = strings.ReplaceAll(k, "\\x03", "\x03")
	k = strings.ReplaceAll(k, "^L", "\x0c")
	k = strings.ReplaceAll(k, "^l", "\x0c")
	k = strings.ReplaceAll(k, "\\x0c", "\x0c")
	return k
}

// Built-in Default Profiles tailored to real-world AI CLI tools
var DefaultProfiles = []AgentProfile{
	{
		ID:         "agy",
		Name:       "Antigravity AI Agent",
		Badge:      "AGY",
		BadgeColor: ColorGreen,
		Buttons: [4]ButtonConfig{
			{Label: "PROCEED", SubText: "[ CONFIRM / OK ]", Color: ColorGreenDark, BorderColor: ColorGreen, LightColor: ColorGreenLight, Action: "proceed", HidKey: "proceed\n"},
			{Label: "GRILL-ME", SubText: "[ ALIGN INTERVIEW ]", Color: ColorCyanDark, BorderColor: ColorCyan, LightColor: ColorCyanLight, Action: "grill-me", HidKey: "/grill-me\n"},
			{Label: "PLAN", SubText: "[ DESIGN / ARCH ]", Color: ColorPurpleDark, BorderColor: ColorPurple, LightColor: ColorPurpleLight, Action: "plan", HidKey: "/plan\n"},
			{Label: "STOP", SubText: "[ ABORT / CTRL+C ]", Color: ColorRedDark, BorderColor: ColorRed, LightColor: ColorRedLight, Action: "stop", HidKey: "\x03"},
		},
	},
	{
		ID:         "claude",
		Name:       "Claude Code CLI",
		Badge:      "CLAUDE",
		BadgeColor: ColorYellow,
		Buttons: [4]ButtonConfig{
			{Label: "PROCEED", SubText: "[ YES / EXECUTE ]", Color: ColorGreenDark, BorderColor: ColorGreen, LightColor: ColorGreenLight, Action: "proceed", HidKey: "y\n"},
			{Label: "COMPACT", SubText: "[ COMPACT HISTORY ]", Color: ColorDarkGray, BorderColor: ColorYellow, LightColor: ColorWhite, Action: "compact", HidKey: "/compact\n"},
			{Label: "COST", SubText: "[ TOKENS & USAGE ]", Color: ColorCyanDark, BorderColor: ColorCyan, LightColor: ColorCyanLight, Action: "cost", HidKey: "/cost\n"},
			{Label: "CANCEL", SubText: "[ ABORT / CTRL+C ]", Color: ColorRedDark, BorderColor: ColorRed, LightColor: ColorRedLight, Action: "stop", HidKey: "\x03"},
		},
	},
	{
		ID:         "dsh",
		Name:       "DeepSeek / Local LLM",
		Badge:      "DSH",
		BadgeColor: ColorCyan,
		Buttons: [4]ButtonConfig{
			{Label: "EXECUTE", SubText: "[ SUBMIT / ENTER ]", Color: ColorGreenDark, BorderColor: ColorGreen, LightColor: ColorGreenLight, Action: "proceed", HidKey: "\n"},
			{Label: "THINK", SubText: "[ TOGGLE R1 COT ]", Color: ColorDarkGray, BorderColor: ColorCyan, LightColor: ColorCyanLight, Action: "think", HidKey: "/think\n"},
			{Label: "CLEAR", SubText: "[ RESET CONTEXT ]", Color: ColorPurpleDark, BorderColor: ColorPurple, LightColor: ColorPurpleLight, Action: "clear", HidKey: "/clear\n"},
			{Label: "STOP", SubText: "[ ABORT / CTRL+C ]", Color: ColorRedDark, BorderColor: ColorRed, LightColor: ColorRedLight, Action: "stop", HidKey: "\x03"},
		},
	},
	{
		ID:         "aider",
		Name:       "Aider Pair Coding",
		Badge:      "AIDER",
		BadgeColor: ColorPurple,
		Buttons: [4]ButtonConfig{
			{Label: "COMMIT", SubText: "[ GIT COMMIT ]", Color: ColorGreenDark, BorderColor: ColorGreen, LightColor: ColorGreenLight, Action: "commit", HidKey: "/commit\n"},
			{Label: "UNDO", SubText: "[ REVERT CHANGE ]", Color: ColorDarkGray, BorderColor: ColorYellow, LightColor: ColorWhite, Action: "undo", HidKey: "/undo\n"},
			{Label: "TEST", SubText: "[ RUN TEST SUITE ]", Color: ColorCyanDark, BorderColor: ColorCyan, LightColor: ColorCyanLight, Action: "test", HidKey: "/test\n"},
			{Label: "CANCEL", SubText: "[ ABORT / CTRL+C ]", Color: ColorRedDark, BorderColor: ColorRed, LightColor: ColorRedLight, Action: "stop", HidKey: "\x03"},
		},
	},
	{
		ID:         "shell",
		Name:       "Developer Terminal",
		Badge:      "SHELL",
		BadgeColor: ColorGray,
		Buttons: [4]ButtonConfig{
			{Label: "ENTER", SubText: "[ RETURN KEY ]", Color: ColorGreenDark, BorderColor: ColorGreen, LightColor: ColorGreenLight, Action: "enter", HidKey: "\n"},
			{Label: "TAB", SubText: "[ AUTO COMPLETE ]", Color: ColorCyanDark, BorderColor: ColorCyan, LightColor: ColorCyanLight, Action: "tab", HidKey: "\t"},
			{Label: "CLEAR", SubText: "[ CTRL+L CLEAR ]", Color: ColorDarkGray, BorderColor: ColorYellow, LightColor: ColorWhite, Action: "clear", HidKey: "\x0c"},
			{Label: "SIGINT", SubText: "[ CTRL+C INTERRUPT ]", Color: ColorRedDark, BorderColor: ColorRed, LightColor: ColorRedLight, Action: "stop", HidKey: "\x03"},
		},
	},
}

var AvailableProfiles = DefaultProfiles

func loadProfilesFromDisk() {
	paths := []string{"/mnt/sdcard/cockpit_agents.json", "/etc/cockpit_agents.json"}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var cfg JsonCockpitConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			log.Printf("[!] JSON parse error in %s: %v", p, err)
			continue
		}
		if len(cfg.Profiles) > 0 {
			var loaded []AgentProfile
			for _, jp := range cfg.Profiles {
				bCol := mapBadgeColor(jp.Color)
				prof := AgentProfile{
					ID:         jp.ID,
					Name:       jp.Name,
					Badge:      jp.Badge,
					BadgeColor: bCol,
				}
				for i := 0; i < 4; i++ {
					jb := jp.Buttons[i]
					bg, bdr, lit := mapColorName(jb.Color)
					prof.Buttons[i] = ButtonConfig{
						Label:       jb.Label,
						SubText:     jb.SubText,
						Color:       bg,
						BorderColor: bdr,
						LightColor:  lit,
						Action:      jb.Action,
						HidKey:      normalizeHidKey(jb.Key),
					}
				}
				loaded = append(loaded, prof)
			}
			AvailableProfiles = loaded
			log.Printf("[+] Loaded %d custom AI agent profiles from %s", len(AvailableProfiles), p)
			return
		}
	}

	// Auto-generate template file on SD card if missing
	templateCfg := JsonCockpitConfig{
		ActiveProfile: "agy",
		Profiles: []JsonAgentProfile{
			{
				ID: "agy", Name: "Antigravity AI Agent", Badge: "AGY", Color: "green",
				Buttons: [4]JsonButtonConfig{
					{Label: "PROCEED", SubText: "[ CONFIRM / OK ]", Action: "proceed", Color: "green", Key: "proceed\\n"},
					{Label: "GRILL-ME", SubText: "[ ALIGN INTERVIEW ]", Action: "grill-me", Color: "cyan", Key: "/grill-me\\n"},
					{Label: "PLAN", SubText: "[ DESIGN / ARCH ]", Action: "plan", Color: "purple", Key: "/plan\\n"},
					{Label: "STOP", SubText: "[ ABORT / CTRL+C ]", Action: "stop", Color: "red", Key: "^C"},
				},
			},
			{
				ID: "claude", Name: "Claude Code CLI", Badge: "CLAUDE", Color: "yellow",
				Buttons: [4]JsonButtonConfig{
					{Label: "PROCEED", SubText: "[ YES / EXECUTE ]", Action: "proceed", Color: "green", Key: "y\\n"},
					{Label: "COMPACT", SubText: "[ COMPACT HISTORY ]", Action: "compact", Color: "yellow", Key: "/compact\\n"},
					{Label: "COST", SubText: "[ TOKENS & USAGE ]", Action: "cost", Color: "cyan", Key: "/cost\\n"},
					{Label: "CANCEL", SubText: "[ ABORT / CTRL+C ]", Action: "stop", Color: "red", Key: "^C"},
				},
			},
			{
				ID: "dsh", Name: "DeepSeek / Local LLM", Badge: "DSH", Color: "cyan",
				Buttons: [4]JsonButtonConfig{
					{Label: "EXECUTE", SubText: "[ SUBMIT / ENTER ]", Action: "proceed", Color: "green", Key: "\\n"},
					{Label: "THINK", SubText: "[ TOGGLE R1 COT ]", Action: "think", Color: "cyan", Key: "/think\\n"},
					{Label: "CLEAR", SubText: "[ RESET CONTEXT ]", Action: "clear", Color: "purple", Key: "/clear\\n"},
					{Label: "STOP", SubText: "[ ABORT / CTRL+C ]", Action: "stop", Color: "red", Key: "^C"},
				},
			},
			{
				ID: "aider", Name: "Aider Pair Coding", Badge: "AIDER", Color: "purple",
				Buttons: [4]JsonButtonConfig{
					{Label: "COMMIT", SubText: "[ GIT COMMIT ]", Action: "commit", Color: "green", Key: "/commit\\n"},
					{Label: "UNDO", SubText: "[ REVERT CHANGE ]", Action: "undo", Color: "yellow", Key: "/undo\\n"},
					{Label: "TEST", SubText: "[ RUN TEST SUITE ]", Action: "test", Color: "cyan", Key: "/test\\n"},
					{Label: "CANCEL", SubText: "[ ABORT / CTRL+C ]", Action: "stop", Color: "red", Key: "^C"},
				},
			},
			{
				ID: "shell", Name: "Developer Terminal", Badge: "SHELL", Color: "gray",
				Buttons: [4]JsonButtonConfig{
					{Label: "ENTER", SubText: "[ RETURN KEY ]", Action: "enter", Color: "green", Key: "\\n"},
					{Label: "TAB", SubText: "[ AUTO COMPLETE ]", Action: "tab", Color: "cyan", Key: "\\t"},
					{Label: "CLEAR", SubText: "[ CTRL+L CLEAR ]", Action: "clear", Color: "yellow", Key: "^L"},
					{Label: "SIGINT", SubText: "[ CTRL+C INTERRUPT ]", Action: "stop", Color: "red", Key: "^C"},
				},
			},
		},
	}
	if tmplBytes, err := json.MarshalIndent(templateCfg, "", "  "); err == nil {
		_ = os.WriteFile("/mnt/sdcard/cockpit_agents.json", tmplBytes, 0644)
	}
}

type CockpitUI struct {
	mu          sync.Mutex
	fbFile      *os.File
	fbMem16     []uint16
	fbMem32     []uint32
	backBuf16   []uint16
	backBuf32   []uint32
	is32bpp     bool
	mapSize     int
	vinfo       FbVarScreeninfo
	metrics     SystemMetrics
	profileIdx  int
	currentView int
	badgeFlash  bool
}

func loadSavedProfile() int {
	paths := []string{"/mnt/sdcard/cockpit_profile", "/tmp/cockpit_profile"}
	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			id := strings.ToLower(strings.TrimSpace(string(data)))
			for idx, prof := range AvailableProfiles {
				if prof.ID == id || strings.ToLower(prof.Badge) == id {
					return idx
				}
			}
		}
	}
	return 0 // default first profile
}

func saveCurrentProfile(idx int) {
	if idx < 0 || idx >= len(AvailableProfiles) {
		return
	}
	id := AvailableProfiles[idx].ID
	paths := []string{"/tmp/cockpit_profile", "/mnt/sdcard/cockpit_profile"}
	for _, p := range paths {
		_ = os.WriteFile(p, []byte(id+"\n"), 0644)
	}
}

func color565To32(c uint16) uint32 {
	r := uint32((c>>11)&0x1f) * 255 / 31
	g := uint32((c>>5)&0x3f) * 255 / 63
	b := uint32(c&0x1f) * 255 / 31
	return 0xff000000 | (r << 16) | (g << 8) | b
}

func NewCockpitUI() (*CockpitUI, error) {
	fb, err := os.OpenFile("/dev/fb0", os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("open /dev/fb0 failed: %v", err)
	}

	var vinfo FbVarScreeninfo
	syscall.Syscall(syscall.SYS_IOCTL, fb.Fd(), FBIOGET_VSCREENINFO, uintptr(unsafe.Pointer(&vinfo)))
	vinfo.Activate = FB_ACTIVATE_NOW | FB_ACTIVATE_FORCE
	vinfo.Xoffset = 0
	vinfo.Yoffset = 0

	bpp := vinfo.BitsPerPixel
	if bpp == 0 {
		bpp = 32
	}
	bytesPerPixel := int(bpp / 8)
	if bytesPerPixel < 2 {
		bytesPerPixel = 2
	}

	totalPixels := ScreenWidth * ScreenHeight
	mapSize := totalPixels * bytesPerPixel

	fbBytes, err := syscall.Mmap(int(fb.Fd()), 0, mapSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		fb.Close()
		return nil, fmt.Errorf("mmap /dev/fb0 failed: %v", err)
	}

	ui := &CockpitUI{
		fbFile:  fb,
		vinfo:   vinfo,
		is32bpp: (bpp == 32),
		mapSize: mapSize,
		metrics: SystemMetrics{
			StreamFps:  "IDLE",
			LastAction: "READY FOR INPUT",
		},
		profileIdx:  loadSavedProfile(),
		currentView: ViewButtons,
	}

	if ui.is32bpp {
		ui.fbMem32 = unsafe.Slice((*uint32)(unsafe.Pointer(&fbBytes[0])), totalPixels)
		ui.backBuf32 = make([]uint32, totalPixels)
	} else {
		ui.fbMem16 = unsafe.Slice((*uint16)(unsafe.Pointer(&fbBytes[0])), totalPixels)
		ui.backBuf16 = make([]uint16, totalPixels)
	}

	return ui, nil
}

func (ui *CockpitUI) DrawPixel(x, y int, color uint16) {
	if x < 0 || x >= ScreenWidth || y < 0 || y >= ScreenHeight {
		return
	}
	var idx int
	if ui.vinfo.Xres == 600 && ui.vinfo.Yres == 1600 {
		idx = x*600 + (599 - y)
	} else {
		idx = y*1600 + x
	}
	if idx < 0 || idx >= ScreenWidth*ScreenHeight {
		return
	}
	if ui.is32bpp {
		ui.backBuf32[idx] = color565To32(color)
	} else {
		ui.backBuf16[idx] = color
	}
}

func (ui *CockpitUI) Clear(color uint16) {
	if ui.is32bpp {
		c32 := color565To32(color)
		for i := range ui.backBuf32 {
			ui.backBuf32[i] = c32
		}
	} else {
		for i := range ui.backBuf16 {
			ui.backBuf16[i] = color
		}
	}
}

func (ui *CockpitUI) DrawRect(x, y, w, h int, color uint16) {
	c32 := color565To32(color)
	isPortrait := (ui.vinfo.Xres == 600 && ui.vinfo.Yres == 1600)

	if isPortrait {
		for cx := x; cx < x+w && cx < ScreenWidth; cx++ {
			if cx < 0 {
				continue
			}
			colBase := cx * 600
			for cy := y; cy < y+h && cy < ScreenHeight; cy++ {
				if cy < 0 {
					continue
				}
				idx := colBase + (599 - cy)
				if ui.is32bpp {
					ui.backBuf32[idx] = c32
				} else {
					ui.backBuf16[idx] = color
				}
			}
		}
	} else {
		for cy := y; cy < y+h && cy < ScreenHeight; cy++ {
			if cy < 0 {
				continue
			}
			rowBase := cy * 1600
			for cx := x; cx < x+w && cx < ScreenWidth; cx++ {
				if cx < 0 {
					continue
				}
				idx := rowBase + cx
				if ui.is32bpp {
					ui.backBuf32[idx] = c32
				} else {
					ui.backBuf16[idx] = color
				}
			}
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
	'v': {0x0E, 0x10, 0x20, 0x10, 0x0E}, // down arrow
}

func (ui *CockpitUI) DrawText(x, y int, text string, scale int, color uint16) {
	currX := x
	for _, ch := range text {
		chUpper := unicode.ToUpper(ch)
		glyph, ok := font5x7[chUpper]
		if ch == 'v' {
			glyph = font5x7['v']
			ok = true
		}
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

// Render the 4 physical tactile macro buttons (ViewButtons)
func (ui *CockpitUI) renderButtons(activeButton int) {
	prof := AvailableProfiles[ui.profileIdx]

	// 1. Right Action Bar Container (X: 1200..1600)
	ui.DrawRect(1200, 0, 400, ScreenHeight, ColorDarkGray)
	ui.DrawRect(1198, 0, 3, ScreenHeight, ColorDivider) // Divider border

	// 2. Top Header HUD (Y: 0..44)
	clockStr := time.Now().Format("15:04:05")
	ui.DrawText(1215, 14, clockStr, 2, ColorWhite)

	// Interactive Profile Mode Pill Badge (X: 1375..1585, Y: 6..38)
	badgeBorder := prof.BadgeColor
	badgeFill := ColorDarkGray
	badgeText := fmt.Sprintf("[%s: MENU v]", prof.Badge)
	if ui.badgeFlash {
		badgeBorder = ColorWhite
		badgeFill = prof.BadgeColor
	}
	ui.DrawRect(1370, 6, 215, 32, badgeFill)
	ui.DrawFrame(1370, 6, 215, 32, 2, badgeBorder)
	textWidth := len(badgeText) * 12
	badgeX := 1370 + (215-textWidth)/2
	ui.DrawText(badgeX, 14, badgeText, 2, ColorWhite)

	// 3. Render 4 Dynamic Tactile Buttons
	yOffsets := []int{46, 166, 286, 406}
	for i := 0; i < 4; i++ {
		btn := prof.Buttons[i]
		btnY := yOffsets[i]
		bgColor := btn.Color
		borderColor := btn.BorderColor
		labelColor := ColorWhite

		if activeButton == (i + 1) {
			bgColor = ColorWhite
			borderColor = btn.LightColor
			labelColor = ColorBlack
		}

		ui.DrawRect(1220, btnY, 360, 108, bgColor)
		ui.DrawFrame(1220, btnY, 360, 108, 3, borderColor)

		// Center main label (scale 3, 18px per char)
		labelW := len(btn.Label) * 18
		labelX := 1400 - labelW/2
		ui.DrawText(labelX, btnY+20, btn.Label, 3, labelColor)

		// Center subtitle (scale 2, 12px per char)
		subW := len(btn.SubText) * 12
		subX := 1400 - subW/2
		ui.DrawText(subX, btnY+66, btn.SubText, 2, btn.LightColor)
	}

	// 4. Footer Telemetry & Tracker (Y: 524..598)
	streamStr := fmt.Sprintf("FPS: %s  RAM: %dM", ui.metrics.StreamFps, ui.metrics.RamUsedMB)
	streamColor := ColorYellow
	if strings.Contains(ui.metrics.StreamFps, "FPS") {
		streamColor = ColorCyan
	}
	ui.DrawText(1240, 532, streamStr, 1, streamColor)

	lastStr := ui.metrics.LastAction
	if ui.metrics.LastActionTime != "" {
		lastStr = fmt.Sprintf("LAST: %s (%s)", ui.metrics.LastAction, ui.metrics.LastActionTime)
	}
	ui.DrawText(1240, 552, lastStr, 1, ColorGreenLight)

	footerLabel := fmt.Sprintf("%s COCKPIT • V553 USB-HID MACRO", prof.Badge)
	ui.DrawText(1240, 574, footerLabel, 1, ColorDivider)
}

// Render the on-device touch-driven Agent Profile Selection Menu (ViewMenu)
func (ui *CockpitUI) renderMenu() {
	// Container
	ui.DrawRect(1200, 0, 400, ScreenHeight, ColorDarkGray)
	ui.DrawRect(1198, 0, 3, ScreenHeight, ColorDivider)

	// Menu Header (Y: 0..48)
	ui.DrawText(1220, 12, "SELECT AI AGENT PROFILE", 2, ColorCyan)
	ui.DrawText(1220, 34, "Tap to activate • Zero host daemon", 1, ColorLightGray)
	ui.DrawRect(1200, 48, 400, 2, ColorDivider)

	// List Available Profiles
	totalProfiles := len(AvailableProfiles)
	if totalProfiles > 5 {
		totalProfiles = 5 // Fit up to 5 on screen cleanly
	}

	for i := 0; i < totalProfiles; i++ {
		prof := AvailableProfiles[i]
		tileY := 54 + i*88
		isActive := (i == ui.profileIdx)

		bgColor := ColorBlack
		borderColor := prof.BadgeColor
		if isActive {
			bgColor = ColorDarkGray
			borderColor = ColorYellow
		}

		ui.DrawRect(1220, tileY, 360, 78, bgColor)
		ui.DrawFrame(1220, tileY, 360, 78, 2, borderColor)

		// Badge Pill
		ui.DrawRect(1230, tileY+12, 85, 26, prof.BadgeColor)
		badgeW := len(prof.Badge) * 12
		ui.DrawText(1230+(85-badgeW)/2, tileY+16, prof.Badge, 2, ColorBlack)

		// Profile Title
		nameStr := prof.Name
		if len(nameStr) > 17 {
			nameStr = nameStr[:17]
		}
		ui.DrawText(1325, tileY+16, nameStr, 2, ColorWhite)

		if isActive {
			ui.DrawText(1510, tileY+18, "*ON*", 1, ColorYellow)
		}

		// Button Key Summary (Row 2)
		summary := fmt.Sprintf("%s | %s | %s | %s",
			prof.Buttons[0].Label, prof.Buttons[1].Label, prof.Buttons[2].Label, prof.Buttons[3].Label)
		if len(summary) > 42 {
			summary = summary[:42]
		}
		ui.DrawText(1230, tileY+50, summary, 1, ColorLightGray)
	}

	// Bottom Cancel / Back Button (Y: 518..578)
	ui.DrawRect(1220, 524, 360, 56, ColorRedDark)
	ui.DrawFrame(1220, 524, 360, 56, 2, ColorRed)
	cancelText := "[ < BACK / CLOSE ]"
	cW := len(cancelText) * 18
	ui.DrawText(1400-cW/2, 542, cancelText, 3, ColorWhite)
}

func (ui *CockpitUI) Flip() {
	isPortrait := (ui.vinfo.Xres == 600 && ui.vinfo.Yres == 1600)
	if isPortrait {
		startIdx := 1198 * 600
		endIdx := 1600 * 600
		if ui.is32bpp {
			copy(ui.fbMem32[startIdx:endIdx], ui.backBuf32[startIdx:endIdx])
		} else {
			copy(ui.fbMem16[startIdx:endIdx], ui.backBuf16[startIdx:endIdx])
		}
	} else {
		for y := 0; y < ScreenHeight; y++ {
			startIdx := y*1600 + 1198
			endIdx := y*1600 + 1600
			if ui.is32bpp {
				copy(ui.fbMem32[startIdx:endIdx], ui.backBuf32[startIdx:endIdx])
			} else {
				copy(ui.fbMem16[startIdx:endIdx], ui.backBuf16[startIdx:endIdx])
			}
		}
	}
}

func (ui *CockpitUI) RenderCurrentView(activeButton int) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	if ui.currentView == ViewMenu {
		ui.renderMenu()
	} else {
		ui.renderButtons(activeButton)
	}
	ui.Flip()
}

func (ui *CockpitUI) ToggleMenu() {
	ui.mu.Lock()
	if ui.currentView == ViewMenu {
		ui.currentView = ViewButtons
	} else {
		ui.currentView = ViewMenu
	}
	if ui.currentView == ViewMenu {
		ui.renderMenu()
	} else {
		ui.renderButtons(0)
	}
	ui.Flip()
	ui.mu.Unlock()
	log.Printf("[UI] View switched to: %v", ui.currentView)
}

func (ui *CockpitUI) CloseMenu() {
	ui.mu.Lock()
	ui.currentView = ViewButtons
	ui.renderButtons(0)
	ui.Flip()
	ui.mu.Unlock()
}

func (ui *CockpitUI) SelectProfile(idx int) {
	if idx < 0 || idx >= len(AvailableProfiles) {
		return
	}
	ui.mu.Lock()
	ui.profileIdx = idx
	ui.currentView = ViewButtons
	ui.badgeFlash = true
	prof := AvailableProfiles[idx]
	saveCurrentProfile(idx)
	ui.metrics.LastAction = fmt.Sprintf("AGENT: %s", prof.Badge)
	ui.metrics.LastActionTime = time.Now().Format("15:04:05")
	ui.renderButtons(0)
	ui.Flip()
	ui.mu.Unlock()

	log.Printf("[AGENT SWITCH] Selected: %s (%s)", prof.Badge, prof.Name)

	go func() {
		time.Sleep(250 * time.Millisecond)
		ui.mu.Lock()
		ui.badgeFlash = false
		if ui.currentView == ViewButtons {
			ui.renderButtons(0)
			ui.Flip()
		}
		ui.mu.Unlock()
	}()
}

func (ui *CockpitUI) CycleProfile() {
	ui.mu.Lock()
	ui.profileIdx = (ui.profileIdx + 1) % len(AvailableProfiles)
	ui.badgeFlash = true
	prof := AvailableProfiles[ui.profileIdx]
	saveCurrentProfile(ui.profileIdx)
	ui.metrics.LastAction = fmt.Sprintf("AGENT: %s", prof.Badge)
	ui.metrics.LastActionTime = time.Now().Format("15:04:05")
	ui.renderButtons(0)
	ui.Flip()
	ui.mu.Unlock()

	log.Printf("[AGENT CYCLE] Cycled to: %s (%s)", prof.Badge, prof.Name)

	go func() {
		time.Sleep(250 * time.Millisecond)
		ui.mu.Lock()
		ui.badgeFlash = false
		if ui.currentView == ViewButtons {
			ui.renderButtons(0)
			ui.Flip()
		}
		ui.mu.Unlock()
	}()
}

func (ui *CockpitUI) SetProfile(name string) {
	nameLower := strings.ToLower(strings.TrimSpace(name))
	for idx, prof := range AvailableProfiles {
		if prof.ID == nameLower || strings.ToLower(prof.Badge) == nameLower {
			ui.SelectProfile(idx)
			return
		}
	}
	log.Printf("[!] Unknown profile requested: %s", name)
}

func (ui *CockpitUI) FlashFeedback(action string) {
	ui.mu.Lock()
	ui.metrics.LastAction = strings.ToUpper(action)
	ui.metrics.LastActionTime = time.Now().Format("15:04:05")

	prof := AvailableProfiles[ui.profileIdx]
	activeBtn := 0
	actionLower := strings.ToLower(action)

	for i := 0; i < 4; i++ {
		if strings.ToLower(prof.Buttons[i].Action) == actionLower ||
			strings.ToLower(prof.Buttons[i].Label) == actionLower {
			activeBtn = i + 1
			break
		}
	}

	if activeBtn == 0 {
		switch actionLower {
		case "proceed", "ready", "y", "execute", "commit", "enter":
			activeBtn = 1
		case "grill-me", "grill", "compact", "think", "undo", "tab":
			activeBtn = 2
		case "plan", "cost", "clear", "test":
			activeBtn = 3
		case "ctrl+c", "stop", "cancel", "sigint":
			activeBtn = 4
		}
	}

	if ui.currentView == ViewButtons {
		ui.renderButtons(activeBtn)
		ui.Flip()
	}
	ui.mu.Unlock()

	go func() {
		time.Sleep(150 * time.Millisecond)
		ui.mu.Lock()
		if ui.currentView == ViewButtons {
			ui.renderButtons(0)
			ui.Flip()
		}
		ui.mu.Unlock()
	}()
}

func (ui *CockpitUI) StartTelemetryCollector() {
	go func() {
		ticker := time.NewTicker(1000 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			temp := float32(0.0)
			if data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp"); err == nil {
				var raw int
				fmt.Sscanf(string(data), "%d", &raw)
				temp = float32(raw) / 1000.0
			}

			ramUsed := 0
			ramTotal := 64
			if data, err := os.ReadFile("/proc/meminfo"); err == nil {
				lines := strings.Split(string(data), "\n")
				var totalKB, freeKB, buffersKB, cachedKB int
				for _, line := range lines {
					if strings.HasPrefix(line, "MemTotal:") {
						fmt.Sscanf(line, "MemTotal: %d", &totalKB)
					} else if strings.HasPrefix(line, "MemFree:") {
						fmt.Sscanf(line, "MemFree: %d", &freeKB)
					} else if strings.HasPrefix(line, "Buffers:") {
						fmt.Sscanf(line, "Buffers: %d", &buffersKB)
					} else if strings.HasPrefix(line, "Cached:") {
						fmt.Sscanf(line, "Cached: %d", &cachedKB)
					}
				}
				ramTotal = totalKB / 1024
				usedKB := totalKB - (freeKB + buffersKB + cachedKB)
				ramUsed = usedKB / 1024
			}

			streamFps := "STANDBY"
			if data, err := os.ReadFile("/tmp/stream_fps"); err == nil {
				fps := strings.TrimSpace(string(data))
				if fps != "" && fps != "0" {
					streamFps = fmt.Sprintf("%s FPS", fps)
				}
			}

			ui.mu.Lock()
			ui.metrics.TempC = temp
			ui.metrics.RamTotalMB = ramTotal
			ui.metrics.RamUsedMB = ramUsed
			ui.metrics.StreamFps = streamFps
			if ui.currentView == ViewButtons {
				ui.renderButtons(0)
				ui.Flip()
			}
			ui.mu.Unlock()
		}
	}()
}

var (
	hidMu   sync.Mutex
	gHidFile *os.File
)

func getHidDevice() (*os.File, error) {
	hidMu.Lock()
	defer hidMu.Unlock()
	if gHidFile != nil {
		return gHidFile, nil
	}
	f, err := os.OpenFile("/dev/hidg0", os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}
	gHidFile = f
	return gHidFile, nil
}

// Low-level USB HID Keyboard Scancode Injection Engine (/dev/hidg0)
func injectHidKeys(hid *os.File, keys []byte) {
	for _, k := range keys {
		press := []byte{0, 0, k, 0, 0, 0, 0, 0}
		release := []byte{0, 0, 0, 0, 0, 0, 0, 0}
		hid.Write(press)
		time.Sleep(15 * time.Millisecond)
		hid.Write(release)
		time.Sleep(15 * time.Millisecond)
	}
}

func injectHidCombo(hid *os.File, mod byte, key byte) {
	press := []byte{mod, 0, key, 0, 0, 0, 0, 0}
	release := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	hid.Write(press)
	time.Sleep(15 * time.Millisecond)
	hid.Write(release)
	time.Sleep(15 * time.Millisecond)
}

func injectHidEnter(hid *os.File, hadPrecedingChars bool) {
	if hadPrecedingChars {
		// Settle pause: allow terminal, shell, and interactive CLI (e.g. agy, ink, readline)
		// to process preceding characters and settle its input state before submitting
		time.Sleep(200 * time.Millisecond)
	}
	press := []byte{0, 0, 0x28, 0, 0, 0, 0, 0}
	release := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	hid.Write(press)
	time.Sleep(50 * time.Millisecond)
	hid.Write(release)
	time.Sleep(30 * time.Millisecond)
	log.Printf("[HID] Enter scancode (0x28) injected (hadPreceding=%v, hold=50ms)", hadPrecedingChars)
}

func injectHidString(hid *os.File, str string) {
	typedCount := 0
	for i := 0; i < len(str); i++ {
		ch := str[i]
		switch {
		case ch == '\x03': // Ctrl+C
			injectHidCombo(hid, 0x01, 0x06)
		case ch == '\x0c': // Ctrl+L
			injectHidCombo(hid, 0x01, 0x0F)
		case ch == '\x1b': // Escape
			injectHidKeys(hid, []byte{0x29})
		case ch == '\n' || ch == '\r': // Enter / Return
			injectHidEnter(hid, typedCount > 0)
			typedCount = 0
		case ch == '\t': // Tab
			injectHidKeys(hid, []byte{0x2B})
			typedCount++
		case ch == ' ': // Space
			injectHidKeys(hid, []byte{0x2C})
			typedCount++
		case ch == '-': // Minus
			injectHidKeys(hid, []byte{0x2D})
			typedCount++
		case ch == '_': // Underscore (Shift + Minus)
			injectHidCombo(hid, 0x02, 0x2D)
			typedCount++
		case ch == '/': // Slash
			injectHidKeys(hid, []byte{0x38})
			typedCount++
		case ch == '?': // Question (Shift + Slash)
			injectHidCombo(hid, 0x02, 0x38)
			typedCount++
		case ch == '.': // Dot
			injectHidKeys(hid, []byte{0x37})
			typedCount++
		case ch == ':': // Colon (Shift + Semicolon)
			injectHidCombo(hid, 0x02, 0x33)
			typedCount++
		case ch == ';': // Semicolon
			injectHidKeys(hid, []byte{0x33})
			typedCount++
		case ch == '=': // Equal
			injectHidKeys(hid, []byte{0x2E})
			typedCount++
		case ch == '+': // Plus (Shift + Equal)
			injectHidCombo(hid, 0x02, 0x2E)
			typedCount++
		case ch == '!': // Exclamation (Shift + 1)
			injectHidCombo(hid, 0x02, 0x1E)
			typedCount++
		case ch == '@': // At (Shift + 2)
			injectHidCombo(hid, 0x02, 0x1F)
			typedCount++
		case ch == '#': // Hash (Shift + 3)
			injectHidCombo(hid, 0x02, 0x20)
			typedCount++
		case ch == '$': // Dollar (Shift + 4)
			injectHidCombo(hid, 0x02, 0x21)
			typedCount++
		case ch == '%': // Percent (Shift + 5)
			injectHidCombo(hid, 0x02, 0x22)
			typedCount++
		case ch == '^': // Caret (Shift + 6)
			injectHidCombo(hid, 0x02, 0x23)
			typedCount++
		case ch == '&': // Ampersand (Shift + 7)
			injectHidCombo(hid, 0x02, 0x24)
			typedCount++
		case ch == '*': // Asterisk (Shift + 8)
			injectHidCombo(hid, 0x02, 0x25)
			typedCount++
		case ch == '(': // Left Paren (Shift + 9)
			injectHidCombo(hid, 0x02, 0x26)
			typedCount++
		case ch == ')': // Right Paren (Shift + 0)
			injectHidCombo(hid, 0x02, 0x27)
			typedCount++
		case ch == '~': // Tilde (Shift + Grave)
			injectHidCombo(hid, 0x02, 0x35)
			typedCount++
		case ch >= 'a' && ch <= 'z':
			sc := byte(0x04 + (ch - 'a'))
			injectHidKeys(hid, []byte{sc})
			typedCount++
		case ch >= 'A' && ch <= 'Z':
			sc := byte(0x04 + (ch - 'A'))
			injectHidCombo(hid, 0x02, sc) // LeftShift + char
			typedCount++
		case ch >= '1' && ch <= '9':
			sc := byte(0x1E + (ch - '1'))
			injectHidKeys(hid, []byte{sc})
			typedCount++
		case ch == '0':
			injectHidKeys(hid, []byte{0x27})
			typedCount++
		default:
			// ignore unsupported
		}
	}
}

// Emits tactile keystrokes strictly over USB HID keyboard interface (/dev/hidg0)
// Absolutely ZERO host network daemons or companion software required.
func sendKeystroke(btn ButtonConfig) {
	// Guard: Ensure action commands terminate with Enter unless it is a control combo like Ctrl+C
	keyToSend := btn.HidKey
	if (btn.Action == "proceed" || btn.Action == "ready" || btn.Action == "enter") &&
		!strings.HasSuffix(keyToSend, "\n") && !strings.HasSuffix(keyToSend, "\r") {
		keyToSend += "\n"
	}

	log.Printf("[touch_cockpit] Triggering Action: %s (Key: %q)", btn.Action, keyToSend)

	// Local device log for on-target debug inspection
	if f, err := os.OpenFile("/tmp/cockpit_event.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
		fmt.Fprintf(f, "%s\n", btn.Action)
		f.Close()
	}

	// Direct hardware USB HID Keyboard injection into host PC via /dev/hidg0
	hid, err := getHidDevice()
	if err != nil {
		log.Printf("[-] /dev/hidg0 open error: %v (is USB ConfigFS gadget active?)", err)
		return
	}

	hidMu.Lock()
	defer hidMu.Unlock()

	log.Printf("[+] Injected USB HID hardware scancodes for [%s] -> /dev/hidg0", btn.Action)
	injectHidString(hid, keyToSend)
	log.Printf("[+] Finished USB HID injection for [%s]", btn.Action)
}

func findTouchDevice() string {
	for i := 0; i < 5; i++ {
		devPath := fmt.Sprintf("/dev/input/event%d", i)
		namePath := fmt.Sprintf("/sys/class/input/event%d/device/name", i)
		if data, err := os.ReadFile(namePath); err == nil {
			name := strings.ToLower(string(data))
			log.Printf("[touch_cockpit] Found input event%d: %s", i, strings.TrimSpace(name))
			if strings.Contains(name, "goodix") || strings.Contains(name, "gt9") || strings.Contains(name, "touch") || strings.Contains(name, "ctp") {
				log.Printf("[touch_cockpit] Auto-selected touch device: %s (%s)", devPath, strings.TrimSpace(name))
				return devPath
			}
		}
	}
	log.Println("[touch_cockpit] Defaulting to /dev/input/event0")
	return "/dev/input/event0"
}

func main() {
	signal.Ignore(syscall.SIGHUP)

	// Logging initialization
	logWriters := []io.Writer{os.Stderr}
	if f, err := os.OpenFile("/tmp/touch_cockpit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
		logWriters = append(logWriters, f)
	}
	if c, err := os.OpenFile("/dev/console", os.O_WRONLY, 0); err == nil {
		logWriters = append(logWriters, c)
	}
	log.SetOutput(io.MultiWriter(logWriters...))

	log.Println("[touch_cockpit] Initializing Standalone USB-HID Multi-Agent Cockpit Controller...")

	// Load custom agents from SD card if present
	loadProfilesFromDisk()

	ui, err := NewCockpitUI()
	if err != nil {
		log.Printf("[!] UI init warning: %v (continuing touch event loop)", err)
	} else {
		ui.mu.Lock()
		ui.Clear(ColorBlack)
		ui.DrawText(60, 260, "MULTI-AGENT AI SMART COCKPIT TERMINAL", 3, ColorCyan)
		ui.DrawText(60, 310, "STANDALONE USB DISPLAY (GUD) + USB HID KEYBOARD", 2, ColorGreenLight)
		ui.mu.Unlock()
		ui.RenderCurrentView(0)
		ui.StartTelemetryCollector()
		log.Printf("[+] Multi-Agent Action Bar active (Initial Agent: %s)", AvailableProfiles[ui.profileIdx].Badge)
	}

	// On-device local loopback command listener (for local camera gestures or shell scripts)
	go func() {
		addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:8889")
		if err != nil {
			return
		}
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			return
		}
		defer conn.Close()
		recvBuf := make([]byte, 128)
		for {
			n, _, err := conn.ReadFrom(recvBuf)
			if err != nil {
				continue
			}
			msg := strings.TrimSpace(string(recvBuf[:n]))
			if strings.HasPrefix(strings.ToLower(msg), "profile ") {
				profName := strings.TrimSpace(msg[8:])
				if profName == "next" || profName == "cycle" {
					if ui != nil {
						ui.CycleProfile()
					}
				} else if ui != nil {
					ui.SetProfile(profName)
				}
			} else if ui != nil {
				go ui.FlashFeedback(msg)
			}
		}
	}()

	touchDev := findTouchDevice()
	evFile, err := os.Open(touchDev)
	if err != nil {
		log.Fatalf("[!] Failed to open touch device %s: %v", touchDev, err)
	}
	defer evFile.Close()

	var (
		rawX            int32 = 0
		rawY            int32 = 0
		fingerDown      bool  = false
		actionTriggered bool  = false
		lastTriggerTime time.Time
	)

	buf := make([]byte, 16)
	for {
		_, err := io.ReadFull(evFile, buf)
		if err != nil {
			log.Fatalf("[-] Error reading event: %v", err)
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
			case ABS_MT_POSITION_Y, ABS_Y:
				rawY = ev.Value
			case ABS_MT_PRESSURE:
				fingerDown = (ev.Value > 0)
			case ABS_MT_TRACKING_ID:
				fingerDown = (ev.Value != -1)
			}
		} else if ev.Type == EV_KEY && ev.Code == BTN_TOUCH {
			fingerDown = (ev.Value == 1)
		} else if ev.Type == EV_SYN && ev.Code == SYN_REPORT {
			if fingerDown {
				var screenX, screenY int
				if rawY > 600 {
					// Goodix GT911 portrait mode (0..600 x 0..1600 -> 1600x600 landscape)
					screenX = int(rawY)
					screenY = 600 - int(rawX)
					if screenY < 0 {
						screenY = int(rawX)
					}
				} else {
					// Goodix GT911 native landscape mode (0..1600 x 0..600)
					screenX = int(rawX)
					screenY = int(rawY)
				}

				// Physical Right 400x600 region (Zone B)
				if screenX >= 1150 {
					log.Printf("[TOUCH] Raw(%d, %d) -> Screen(%d, %d)", rawX, rawY, screenX, screenY)
					if ui != nil && ui.currentView == ViewMenu {
						// MENU VIEW TOUCH DISPATCH
						if !actionTriggered && time.Since(lastTriggerTime) >= 350*time.Millisecond {
							// 1. Check profile item tiles
							if screenY >= 54 && screenY < 54+len(AvailableProfiles)*88 {
								itemIdx := (screenY - 54) / 88
								if itemIdx >= 0 && itemIdx < len(AvailableProfiles) {
									actionTriggered = true
									lastTriggerTime = time.Now()
									ui.SelectProfile(itemIdx)
									continue
								}
							}
							// 2. Check bottom Cancel / Back button
							if screenY >= 518 {
								actionTriggered = true
								lastTriggerTime = time.Now()
								ui.CloseMenu()
								continue
							}
						}
					} else {
						// TACTILE BUTTONS VIEW TOUCH DISPATCH
						// 1. Check Header Mode Pill Badge tapped (Y < 45) -> Opens Selection Menu
						if screenY < 45 && !actionTriggered && time.Since(lastTriggerTime) >= 400*time.Millisecond {
							actionTriggered = true
							lastTriggerTime = time.Now()
							if ui != nil {
								ui.ToggleMenu()
							}
							continue
						}

						// 2. Check 4 tactile action buttons
						btn := 0
						if screenY >= 46 && screenY < 155 {
							btn = 1
						} else if screenY >= 166 && screenY < 275 {
							btn = 2
						} else if screenY >= 286 && screenY < 395 {
							btn = 3
						} else if screenY >= 406 && screenY <= 520 {
							btn = 4
						}

						if btn > 0 && !actionTriggered && time.Since(lastTriggerTime) >= 800*time.Millisecond {
							actionTriggered = true
							lastTriggerTime = time.Now()
							if ui != nil {
								prof := AvailableProfiles[ui.profileIdx]
								btnCfg := prof.Buttons[btn-1]
								log.Printf("[TOUCH HIT] Button %d [%s: %s] -> HID Keystroke", btn, prof.Badge, btnCfg.Label)
								ui.FlashFeedback(btnCfg.Action)
								go sendKeystroke(btnCfg)
							}
						}
					}
				}
			} else {
				actionTriggered = false
			}
		}
	}
}
