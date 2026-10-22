package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const MEMERASE = 0x40084d02

type eraseInfo struct {
	start  uint32
	length uint32
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <image_file> <mtd_device> [--reboot]", os.Args[0])
	}

	imgPath := os.Args[1]
	mtdPath := os.Args[2]
	autoReboot := len(os.Args) >= 4 && os.Args[3] == "--reboot"

	imgData, err := os.ReadFile(imgPath)
	if err != nil {
		log.Fatalf("Failed to read image: %v", err)
	}

	h := sha256.Sum256(imgData)
	fmt.Printf("[*] Image: %s (%d bytes, SHA256: %s)\n", imgPath, len(imgData), hex.EncodeToString(h[:]))

	mtd, err := os.OpenFile(mtdPath, os.O_RDWR|os.O_SYNC, 0)
	if err != nil {
		log.Fatalf("Failed to open %s: %v", mtdPath, err)
	}
	defer mtd.Close()

	// Erase in 64KB blocks
	eraseSize := uint32(65536)
	totalErase := ((uint32(len(imgData)) + eraseSize - 1) / eraseSize) * eraseSize
	fmt.Printf("[*] Erasing %d bytes (in %d blocks of 64KB)...\n", totalErase, totalErase/eraseSize)

	for offset := uint32(0); offset < totalErase; offset += eraseSize {
		ei := eraseInfo{start: offset, length: eraseSize}
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, mtd.Fd(), uintptr(MEMERASE), uintptr(unsafe.Pointer(&ei)))
		if errno != 0 {
			log.Fatalf("Erase failed at offset 0x%x: %v", offset, errno)
		}
		if (offset/eraseSize)%16 == 0 || offset+eraseSize >= totalErase {
			fmt.Printf("\r[*] Erased %d / %d KB...", (offset+eraseSize)/1024, totalErase/1024)
		}
	}
	fmt.Println("\n[*] Erase complete!")

	// Write image
	fmt.Println("[*] Writing image...")
	_, err = mtd.Seek(0, 0)
	if err != nil {
		log.Fatalf("Seek failed: %v", err)
	}

	n, err := mtd.Write(imgData)
	if err != nil || n != len(imgData) {
		log.Fatalf("Write failed: wrote %d of %d bytes: %v", n, len(imgData), err)
	}
	fmt.Printf("[*] Written %d bytes successfully!\n", n)

	// Verify
	fmt.Println("[*] Verifying written data against flash...")
	_, err = mtd.Seek(0, 0)
	if err != nil {
		log.Fatalf("Seek for verify failed: %v", err)
	}

	verifyBuf := make([]byte, len(imgData))
	_, err = io.ReadFull(mtd, verifyBuf)
	if err != nil {
		log.Fatalf("Read for verify failed: %v", err)
	}

	vh := sha256.Sum256(verifyBuf)
	if vh != h {
		log.Fatalf("[!] Verification FAILED! Expected %s, got %s", hex.EncodeToString(h[:]), hex.EncodeToString(vh[:]))
	}

	fmt.Printf("[SUCCESS] Flash Verification PASSED (SHA256: %s)!\n", hex.EncodeToString(vh[:]))

	if autoReboot {
		fmt.Println("[*] Flashing completed cleanly. Triggering kernel reboot in 1 second...")
		time.Sleep(1 * time.Second)
		syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	}
}
