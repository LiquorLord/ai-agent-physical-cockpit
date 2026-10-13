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
)

const (
	FrameWidth  = 1600
	FrameHeight = 600
	BytesPerPix = 2
	FrameSize   = FrameWidth * FrameHeight * BytesPerPix // 1,920,000 bytes
	ListenPort  = ":5900"
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

	fb, err := os.OpenFile("/dev/fb0", os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("[!] Failed to open /dev/fb0: %v", err)
	}
	defer fb.Close()

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

			frameCount++
			if frameCount%100 == 0 {
				fmt.Printf("\r[*] Blitted %d frames...", frameCount)
			}
		}

		conn.Close()
		log.Printf("\n[-] Host disconnected after %d frames. Ready for next stream.", frameCount)
	}
}
