package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
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
	ScreenWidth        = 1600
	ScreenHeight       = 600
	BytesPerPix        = 2
	ScreenStride       = ScreenWidth * BytesPerPix // 3200 bytes per line
	FullFrameSize      = ScreenWidth * ScreenHeight * BytesPerPix // 1,920,000 bytes

	DualZoneWidth      = 1200
	DualZoneRowBytes   = DualZoneWidth * BytesPerPix // 2400 bytes per line
	DualZoneFrameSize  = DualZoneWidth * ScreenHeight * BytesPerPix // 1,440,000 bytes

	DualZonePort       = ":5900" // Left 1200x600 Content Viewport
	FullScreenPort     = ":5901" // Full 1600x600 Display

	FBIOGET_VSCREENINFO = 0x4600
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
			time.Sleep(50 * time.Millisecond)
		}
	}()
}

type DisplayServer struct {
	fbFile *os.File
	fbMem  []byte
	vinfo  FbVarScreeninfo
	mu     sync.Mutex
}

func (ds *DisplayServer) panDisplay() {
	syscall.Syscall(syscall.SYS_IOCTL, ds.fbFile.Fd(), FBIOPAN_DISPLAY, uintptr(unsafe.Pointer(&ds.vinfo)))
}

func (ds *DisplayServer) handleDualZoneStream(conn net.Conn) {
	remote := conn.RemoteAddr().String()
	log.Printf("[+] Dual-Zone Stream connected from %s (1200x600 RGB565 -> Columns 0..1200)", remote)
	defer conn.Close()

	frameBuf := make([]byte, DualZoneFrameSize)
	frameCount := 0
	startTime := time.Now()

	for {
		_, err := io.ReadFull(conn, frameBuf)
		if err != nil {
			if err != io.EOF {
				log.Printf("[-] Dual-Zone read error: %v", err)
			}
			break
		}

		ds.mu.Lock()
		// Row-by-row blitting into columns 0..1200 (preserving right 400 columns for Action Bar)
		for y := 0; y < ScreenHeight; y++ {
			destOffset := y * ScreenStride
			srcOffset := y * DualZoneRowBytes
			copy(ds.fbMem[destOffset:destOffset+DualZoneRowBytes], frameBuf[srcOffset:srcOffset+DualZoneRowBytes])
		}
		ds.panDisplay()
		ds.mu.Unlock()

		frameCount++
		if frameCount%100 == 0 {
			elapsed := time.Since(startTime).Seconds()
			fps := float64(frameCount) / elapsed
			fmt.Printf("\r[*] Dual-Zone: blitted %d frames (%.1f FPS)...", frameCount, fps)
		}
	}

	elapsed := time.Since(startTime).Seconds()
	fps := float64(frameCount) / elapsed
	log.Printf("\n[-] Dual-Zone client disconnected after %d frames (avg %.1f FPS). Ready for next stream.", frameCount, fps)
}

func (ds *DisplayServer) handleFullScreenStream(conn net.Conn) {
	remote := conn.RemoteAddr().String()
	log.Printf("[+] Full-Screen Stream connected from %s (1600x600 RGB565)", remote)
	defer conn.Close()

	frameBuf := make([]byte, FullFrameSize)
	frameCount := 0
	startTime := time.Now()

	for {
		_, err := io.ReadFull(conn, frameBuf)
		if err != nil {
			if err != io.EOF {
				log.Printf("[-] Full-Screen read error: %v", err)
			}
			break
		}

		ds.mu.Lock()
		copy(ds.fbMem[0:FullFrameSize], frameBuf[0:FullFrameSize])
		ds.panDisplay()
		ds.mu.Unlock()

		frameCount++
		if frameCount%100 == 0 {
			elapsed := time.Since(startTime).Seconds()
			fps := float64(frameCount) / elapsed
			fmt.Printf("\r[*] Full-Screen: blitted %d frames (%.1f FPS)...", frameCount, fps)
		}
	}

	elapsed := time.Since(startTime).Seconds()
	fps := float64(frameCount) / elapsed
	log.Printf("\n[-] Full-Screen client disconnected after %d frames (avg %.1f FPS). Ready for next stream.", frameCount, fps)
}

func main() {
	log.Printf("[fb_receiver] Starting V553 High-Performance Dual-Zone Framebuffer Receiver...")
	log.Printf("[fb_receiver] Port %s -> Dual-Zone (1200x600, preserving Action Bar)", DualZonePort)
	log.Printf("[fb_receiver] Port %s -> Full-Screen (1600x600)", FullScreenPort)

	// Takeover hardware watchdog
	startWatchdogFeeder()

	fb, err := os.OpenFile("/dev/fb0", os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("[!] Failed to open /dev/fb0: %v", err)
	}
	defer fb.Close()

	// Memory map /dev/fb0 for direct zero-copy hardware framebuffer blitting
	fbMem, err := syscall.Mmap(int(fb.Fd()), 0, FullFrameSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		log.Fatalf("[!] Failed to mmap /dev/fb0: %v", err)
	}
	defer syscall.Munmap(fbMem)
	log.Printf("[+] Hardware Framebuffer successfully mmap'ed (%d bytes)!", len(fbMem))

	var v FbVarScreeninfo
	syscall.Syscall(syscall.SYS_IOCTL, fb.Fd(), FBIOGET_VSCREENINFO, uintptr(unsafe.Pointer(&v)))
	v.Activate = FB_ACTIVATE_NOW | FB_ACTIVATE_FORCE
	v.Xoffset = 0
	v.Yoffset = 0

	ds := &DisplayServer{
		fbFile: fb,
		fbMem:  fbMem,
		vinfo:  v,
	}

	// Setup listeners
	lDual, err := net.Listen("tcp", DualZonePort)
	if err != nil {
		log.Fatalf("[!] Failed to listen on Dual-Zone port %s: %v", DualZonePort, err)
	}
	defer lDual.Close()

	lFull, err := net.Listen("tcp", FullScreenPort)
	if err != nil {
		log.Fatalf("[!] Failed to listen on Full-Screen port %s: %v", FullScreenPort, err)
	}
	defer lFull.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("[*] Terminating fb_receiver...")
		lDual.Close()
		lFull.Close()
		fb.Close()
		os.Exit(0)
	}()

	// Dual-zone accept loop
	go func() {
		for {
			conn, err := lDual.Accept()
			if err != nil {
				return
			}
			ds.handleDualZoneStream(conn)
		}
	}()

	// Full-screen accept loop
	for {
		conn, err := lFull.Accept()
		if err != nil {
			return
		}
		ds.handleFullScreenStream(conn)
	}
}
