package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
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
	FrameWidth          = 1600
	FrameHeight         = 600
	BytesPerPix         = 2
	FrameSize           = FrameWidth * FrameHeight * BytesPerPix // 1,920,000 bytes
	ListenPort          = ":5900"
	FBIOGET_VSCREENINFO = 0x4600
	FBIOPUT_VSCREENINFO = 0x4601
	FBIOPAN_DISPLAY     = 0x4606
	FB_ACTIVATE_NOW     = 0
	FB_ACTIVATE_FORCE   = 128
)

func startWatchdogFeeder() {
	go func() {
		for {
			wdt, err := os.OpenFile("/dev/watchdog", os.O_WRONLY, 0)
			if err == nil {
				log.Println("[+] Hardware Watchdog successfully acquired! Feeding every 2s...")
				for {
					_, err := wdt.Write([]byte("1"))
					if err != nil {
						log.Printf("[-] Watchdog write error: %v", err)
						wdt.Close()
						break
					}
					time.Sleep(2 * time.Second)
				}
			}
			// Retry every 50ms until acquired
			time.Sleep(50 * time.Millisecond)
		}
	}()
}

func main() {
	log.Printf("[fb_receiver] Starting V553 Framebuffer Receiver (%dx%d RGB565)...", FrameWidth, FrameHeight)

	// Start background watchdog takeover immediately
	startWatchdogFeeder()

	fb, err := os.OpenFile("/dev/fb0", os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("[!] Failed to open /dev/fb0: %v", err)
	}
	defer fb.Close()

	// Query screeninfo for page flipping
	var v FbVarScreeninfo
	syscall.Syscall(syscall.SYS_IOCTL, fb.Fd(), FBIOGET_VSCREENINFO, uintptr(unsafe.Pointer(&v)))
	v.Activate = FB_ACTIVATE_NOW | FB_ACTIVATE_FORCE
	v.Xoffset = 0
	v.Yoffset = 0

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	l, err := net.Listen("tcp", ListenPort)
	if err != nil {
		log.Fatalf("[!] Failed to listen on %s: %v", ListenPort, err)
	}
	defer l.Close()

	log.Printf("[*] Listening on TCP %s (Waiting for host stream)...", ListenPort)

	go func() {
		<-sigChan
		log.Println("[*] Terminating fb_receiver...")
		l.Close()
		fb.Close()
		os.Exit(0)
	}()

	frameBuf := make([]byte, FrameSize)

	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-sigChan:
				return
			default:
				continue
			}
		}

		remote := conn.RemoteAddr().String()
		log.Printf("[+] Host stream connected from %s", remote)

		frameCount := 0
		for {
			_, err := io.ReadFull(conn, frameBuf)
			if err != nil {
				if err != io.EOF {
					log.Printf("[-] Read error: %v", err)
				}
				break
			}

			// Seek to beginning and blit full frame to fb0
			_, err = fb.Seek(0, 0)
			if err != nil {
				log.Printf("[-] Seek error: %v", err)
				break
			}

			_, err = fb.Write(frameBuf)
			if err != nil {
				log.Printf("[-] Write error: %v", err)
				break
			}

			// Pan display to trigger hardware page flip on Allwinner DE2.0
			syscall.Syscall(syscall.SYS_IOCTL, fb.Fd(), FBIOPAN_DISPLAY, uintptr(unsafe.Pointer(&v)))

			frameCount++
			if frameCount%100 == 0 {
				fmt.Printf("\r[*] Blitted %d frames...", frameCount)
			}
		}

		conn.Close()
		log.Printf("\n[-] Host disconnected after %d frames. Ready for next stream.", frameCount)
	}
}
