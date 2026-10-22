#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Direct Hardware SPI JEDEC Probe for Allwinner V853 via FEL
Probe the V853 SPI Flash JEDEC ID via FEL direct hardware register access
"""

import subprocess
import time

def readl(addr):
    res = subprocess.check_output(["./sunxi-tools/sunxi-fel", "readl", hex(addr)]).decode().strip()
    return int(res, 16)

def writel(addr, val):
    subprocess.check_call(["./sunxi-tools/sunxi-fel", "writel", hex(addr), hex(val)])

def main():
    print("[*] Testing V853 SPI0 JEDEC ID read via FEL...")

    # 1. CCU Clock & Reset
    # 0x0200196C: SPI_BGR_REG (Bit 0: gate, Bit 16: reset deassert)
    writel(0x0200196C, (1 << 16) | (1 << 0))
    # 0x02001940: SPI0_CLK_REG (Bit 31: enable clock, Source: OSC24M)
    writel(0x02001940, 1 << 31)

    # 2. GPIO Port C Pinmux: PC0..PC5 = Function 4 (SPI0)
    writel(0x02000060, 0x00444444)

    # 3. SPI0 Controller Init
    # Reset and Master Mode Enable
    writel(0x04025004, (1 << 31) | 3) # GCR
    time.sleep(0.01)
    gcr = readl(0x04025004)
    print(f"GCR after reset: {hex(gcr)}")

    # Clock divider
    writel(0x04025024, 0x1002) # CCTL: 24MHz / 6 = 4MHz

    # TCR: Bit 7 (DHB: discard hash burst), Bit 2 (SS_OWN: hardware CS), SS_SEL=0 (CS0), SPOL=0
    # On Allwinner SUN6I SPI, TCR:
    # Bit 31: XCH (Exchange)
    # Bit 7: DHB
    # Bit 2: SS_OWN
    # Bit 0,1: CPOL, CPHA
    writel(0x04025008, 0x00000000)

    # Set counters for 4 bytes transfer (1 byte CMD 0x9F + 3 bytes read)
    writel(0x04025030, 4) # MBC (Master Burst Counter)
    writel(0x04025034, 1) # MTC (Master Transmit Counter)
    writel(0x04025038, 1) # BCC (Master Burst Control Counter: 1 byte write)

    # Write command 0x9F to TXD (0x04025200)
    writel(0x04025200, 0x9F)

    # Start Exchange: TCR |= (1 << 31)
    writel(0x04025008, (1 << 31) | (1 << 7))

    # Wait for XCH to clear
    for _ in range(50):
        tcr = readl(0x04025008)
        if not (tcr & (1 << 31)):
            break
        time.sleep(0.01)

    print(f"TCR after transfer: {hex(tcr)}")
    fsr = readl(0x04025014) # FIFO Status Register
    print(f"FSR (FIFO Status): {hex(fsr)}")

    # Read RX FIFO (0x04025300)
    rx_data = []
    for _ in range(4):
        rx_data.append(readl(0x04025300) & 0xFF)

    print("Received bytes:", [hex(b) for b in rx_data])

if __name__ == "__main__":
    main()
