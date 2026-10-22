# Factory SPI Flasher

Tools for reading and writing the Allwinner V553 factory SPI NOR flash via FEL
(Fastboot/Loader) mode, plus the local patches we carry on top of upstream
`sunxi-tools`.

## Prerequisites

The `flash_tool.go` and `fel_probe.sh` scripts depend on `sunxi-fel`, which is
provided by the upstream [`linux-sunxi/sunxi-tools`](https://github.com/linux-sunxi/sunxi-tools)
project. We do **not** vendor that repository here; clone it yourself and apply
our V553 SPI flash patch on top:

```bash
# 1. Clone upstream sunxi-tools (fel)
git clone https://github.com/linux-sunxi/sunxi-tools.git
cd sunxi-tools

# 2. Apply our local V553 / XM25QHxx SPI support patch
git apply ../sunxi-tools-spiflash.patch

# 3. Build (see upstream README for build dependencies)
make
```

The patch `sunxi-tools-spiflash.patch` adds:

- SPI flash identification for the **XMC XM25QH128C** (JEDEC ID `0x2040`).
- V853/V553 (`soc_id 0x1886`) GPIO / SPI base addresses and SPI0 clock/gate
  offsets for FEL-based SPI programming.

## Contents

| File | Purpose |
| :--- | :--- |
| `flash_tool.go` | MTD `MEMERASE`-based SPI flashing tool (runs on-device) |
| `dump_spiflash.sh` | Dump the factory SPI NOR flash |
| `fel_probe.sh` | Probe the SoC in FEL mode via `sunxi-fel` |
| `sunxi-tools-spiflash.patch` | V553 / XM25QHxx support patch for upstream `sunxi-tools` |
