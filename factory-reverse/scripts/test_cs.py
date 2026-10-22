#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Test CS pin candidate (PC1, PC3, PC5) for SPI Flash on V853
"""

import subprocess
import time

def readl(addr):
    res = subprocess.check_output(["./sunxi-tools/sunxi-fel", "readl", hex(addr)]).decode().strip()
    return int(res, 16)

def writel(addr, val):
    subprocess.check_call(["./sunxi-tools/sunxi-fel", "writel", hex(addr), hex(val)])

def test_cs_pin(cs_pin_num):
    print(f"\n--- Testing PC{cs_pin_num} as Chip Select (CS) ---")
    
    # Reset SPI0 & CCU
    writel(0x0200196C, (1 << 16) | (1 << 0))
    writel(0x02001940, 1 << 31)

    # Configure PC: PC0=CLK (mux 4), PC1=MOSI/DIN (mux 4), PC2=MISO/DOUT (mux 4)
    # Set candidate CS pin as GPIO Output (Function 1)
    cfg0 = 0x00444444
    # Clear candidate pin in cfg0 (bits pin*4 to pin*4+3)
    shift = cs_pin_num * 4
    cfg0 &= ~(0xF << shift)
    cfg0 |= (1 << shift) # Function 1 = Output
    writel(0x02000060, cfg0)

    # Set CS high initially (PC_DATA is at 0x02000070)
    data_addr = 0x02000070
    writel(data_addr, readl(data_addr) | (1 << cs_pin_num))

    # Reset SPI controller
    writel(0x04025004, (1 << 31) | 3)
    time.sleep(0.005)
    writel(0x04025024, 0x1002) # Clock / 6 = 4MHz

    # Pull CS LOW
    writel(data_addr, readl(data_addr) & ~(1 << cs_pin_num))

    # Prepare transfer (MBC=4, MTC=1, BCC=1)
    # TCR: Manual SS_OWN=1, SS_LEVEL=0
    writel(0x04025008, (1 << 2) | (1 << 7))
    writel(0x04025030, 4)
    writel(0x04025034, 1)
    writel(0x04025038, 1)
    writel(0x04025200, 0x9F) # TXD: 0x9F JEDEC ID (at offset 0x200)

    # Start exchange
    writel(0x04025008, (1 << 31) | (1 << 2) | (1 << 7))
    for _ in range(50):
        tcr = readl(0x04025008)
        if not (tcr & (1 << 31)):
            break
        time.sleep(0.005)

    # Pull CS HIGH
    writel(data_addr, readl(data_addr) | (1 << cs_pin_num))

    # Read RX FIFO
    rx = [hex(readl(0x04025300) & 0xFF) for _ in range(4)]
    print(f"Result with PC{cs_pin_num} CS: {rx}")
    if rx != ['0xff', '0xff', '0xff', '0xff'] and rx != ['0x0', '0x0', '0x0', '0x0']:
        print(f"[!!!] SUCCESS! Flash responded on CS PC{cs_pin_num}: {rx}")
        return True
    return False

def main():
    for pin in [1, 3, 5, 4, 0, 2]:
        if test_cs_pin(pin):
            break

if __name__ == "__main__":
    main()
