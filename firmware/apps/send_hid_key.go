package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Standard USB HID Keyboard Usage Codes
var keycodeMap = map[string]byte{
	"enter":     0x28,
	"return":    0x28,
	"esc":       0x29,
	"escape":    0x29,
	"backspace": 0x2A,
	"tab":       0x2B,
	"space":     0x2C,
	"y":         0x1C,
	"n":         0x11,
	"c":         0x06,
	"a":         0x04,
	"b":         0x05,
	"d":         0x07,
	"e":         0x08,
	"f":         0x09,
	"g":         0x0A,
	"h":         0x0B,
	"i":         0x0C,
	"j":         0x0D,
	"k":         0x0E,
	"l":         0x0F,
	"m":         0x10,
	"o":         0x12,
	"p":         0x13,
	"q":         0x14,
	"r":         0x15,
	"s":         0x16,
	"t":         0x17,
	"u":         0x18,
	"v":         0x19,
	"w":         0x1A,
	"x":         0x1B,
	"z":         0x1D,
}

const (
	ModCtrl  = 0x01
	ModShift = 0x02
	ModAlt   = 0x04
	ModMeta  = 0x08
)

func sendReport(fd *os.File, mod byte, key byte) error {
	report := []byte{mod, 0x00, key, 0x00, 0x00, 0x00, 0x00, 0x00}
	if _, err := fd.Write(report); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)

	// Release report (all zeros)
	release := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if _, err := fd.Write(release); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: send_hid_key <key_or_combo>")
		fmt.Println("Examples:")
		fmt.Println("  send_hid_key enter")
		fmt.Println("  send_hid_key y")
		fmt.Println("  send_hid_key ctrl+c")
		fmt.Println("  send_hid_key text \"echo ok\"")
		os.Exit(1)
	}

	devPath := "/dev/hidg0"
	fd, err := os.OpenFile(devPath, os.O_WRONLY, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", devPath, err)
		os.Exit(1)
	}
	defer fd.Close()

	arg := os.Args[1]

	if arg == "text" && len(os.Args) >= 3 {
		text := os.Args[2]
		for _, ch := range text {
			s := strings.ToLower(string(ch))
			if s == " " {
				sendReport(fd, 0, keycodeMap["space"])
			} else if code, ok := keycodeMap[s]; ok {
				var mod byte = 0
				if ch >= 'A' && ch <= 'Z' {
					mod = ModShift
				}
				sendReport(fd, mod, code)
			}
		}
		fmt.Printf("[OK] Injected text: %q\n", text)
		return
	}

	parts := strings.Split(strings.ToLower(arg), "+")
	var mod byte = 0
	var key byte = 0

	for _, p := range parts {
		switch p {
		case "ctrl":
			mod |= ModCtrl
		case "shift":
			mod |= ModShift
		case "alt":
			mod |= ModAlt
		case "meta", "super", "win":
			mod |= ModMeta
		default:
			if code, ok := keycodeMap[p]; ok {
				key = code
			} else {
				fmt.Fprintf(os.Stderr, "Unknown key: %s\n", p)
				os.Exit(1)
			}
		}
	}

	if err := sendReport(fd, mod, key); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send HID report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[OK] HID key injected: %s (mod: 0x%02x, key: 0x%02x)\n", arg, mod, key)
}
