/* SPDX-License-Identifier: MIT */
/*
 * GUD (Generic USB Display) Gadget Daemon for Allwinner V553
 * Emulates a native USB Display Adapter over USB-C using FunctionFS
 * Supports 1600x600 BOE MIPI Display with zero-copy hardware blitting
 *
 * English & Chinese Bilingual Standard / 英汉双语规范
 */

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>
#include <endian.h>
#include <pthread.h>
#include <signal.h>
#include <sys/mman.h>
#include <sys/ioctl.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <linux/types.h>
#include <linux/fb.h>
#include <linux/usb/ch9.h>
#include <linux/usb/functionfs.h>

#define TOTAL_PANEL_WIDTH    1600
#define GUD_VIEWPORT_WIDTH   1200
#define ACTION_BAR_WIDTH     400
#define SCREEN_HEIGHT        600
#define BPP                  2
#define FRAME_SIZE           (TOTAL_PANEL_WIDTH * SCREEN_HEIGHT * BPP) /* 1,920,000 bytes */
#define PANEL_WIDTH          600
#define PANEL_HEIGHT         1600

#define EP0_PATH             "/dev/ffs-gud/ep0"
#define EP1_PATH             "/dev/ffs-gud/ep1"
#define FB_PATH              "/dev/fb0"

static volatile uint32_t g_active_mode_width = GUD_VIEWPORT_WIDTH;

/* GUD Protocol Definitions (from drm/gud.h) */
#define GUD_DISPLAY_MAGIC                       0x1d50614d
#define GUD_DISPLAY_FLAG_STATUS_ON_SET          (1 << 0)
#define GUD_DISPLAY_FLAG_FULL_UPDATE            (1 << 1)

#define GUD_COMPRESSION_LZ4                     (1 << 0)

#define GUD_PIXEL_FORMAT_RGB565                 0x40

#define GUD_CONNECTOR_TYPE_PANEL                0
#define GUD_CONNECTOR_STATUS_CONNECTED          0x01

#define GUD_DISPLAY_MODE_FLAG_PHSYNC            (1 << 0)
#define GUD_DISPLAY_MODE_FLAG_PVSYNC            (1 << 2)
#define GUD_DISPLAY_MODE_FLAG_PREFERRED         (1 << 10)

/* GUD USB Requests */
#define GUD_REQ_GET_STATUS                      0x00
  #define GUD_STATUS_OK                         0x00
  #define GUD_STATUS_BUSY                       0x01
  #define GUD_STATUS_REQUEST_NOT_SUPPORTED      0x02
  #define GUD_STATUS_PROTOCOL_ERROR             0x03
  #define GUD_STATUS_INVALID_PARAMETER          0x04
  #define GUD_STATUS_ERROR                      0x05

#define GUD_REQ_GET_DESCRIPTOR                  0x01
#define GUD_REQ_GET_FORMATS                     0x40
#define GUD_REQ_GET_PROPERTIES                  0x41
#define GUD_REQ_GET_CONNECTORS                  0x50
#define GUD_REQ_GET_CONNECTOR_PROPERTIES        0x51
#define GUD_REQ_SET_CONNECTOR_FORCE_DETECT      0x53
#define GUD_REQ_GET_CONNECTOR_STATUS            0x54
#define GUD_REQ_GET_CONNECTOR_MODES             0x55
#define GUD_REQ_GET_CONNECTOR_EDID              0x56
#define GUD_REQ_SET_BUFFER                      0x60
#define GUD_REQ_SET_STATE_CHECK                 0x61
#define GUD_REQ_SET_STATE_COMMIT                0x62
#define GUD_REQ_SET_CONTROLLER_ENABLE           0x63
#define GUD_REQ_SET_DISPLAY_ENABLE              0x64

/* GUD Protocol Structures */
struct gud_display_descriptor_req {
    __le32 magic;
    __u8 version;
    __le32 flags;
    __u8 compression;
    __le32 max_buffer_size;
    __le32 min_width;
    __le32 max_width;
    __le32 min_height;
    __le32 max_height;
} __attribute__((packed));

struct gud_display_mode_req {
    __le32 clock;
    __le16 hdisplay;
    __le16 hsync_start;
    __le16 hsync_end;
    __le16 htotal;
    __le16 vdisplay;
    __le16 vsync_start;
    __le16 vsync_end;
    __le16 vtotal;
    __le32 flags;
} __attribute__((packed));

struct gud_connector_descriptor_req {
    __u8 connector_type;
    __le32 flags;
} __attribute__((packed));

struct gud_set_buffer_req {
    __le32 x;
    __le32 y;
    __le32 width;
    __le32 height;
    __le32 length;
    __u8 compression;
    __le32 compressed_length;
} __attribute__((packed));

/* FunctionFS Descriptor Types */
struct desc_v2 {
    struct usb_functionfs_descs_head_v2 header;
    __le32 fs_count;
    __le32 hs_count;
    struct {
        struct usb_interface_descriptor intf;
        struct usb_endpoint_descriptor_no_audio bulk_out;
    } __attribute__((packed)) fs_descs;
    struct {
        struct usb_interface_descriptor intf;
        struct usb_endpoint_descriptor_no_audio bulk_out;
    } __attribute__((packed)) hs_descs;
} __attribute__((packed));

struct strings {
    struct usb_functionfs_strings_head header;
    struct {
        __le16 code;
        char str1[sizeof("GUD USB Display")];
    } __attribute__((packed)) tab;
} __attribute__((packed));

/* Global State */
static int g_ep0_fd = -1;
static int g_ep1_fd = -1;
static int g_fb_fd = -1;
static uint16_t *g_fb_mem16 = NULL;
static uint32_t *g_fb_mem32 = NULL;
static bool g_is_32bpp = false;
static size_t g_fb_map_size = 0;
static struct fb_var_screeninfo g_vinfo;
static volatile bool g_running = true;
static volatile bool g_force_full_frame = true;

/* Buffer transfer queue between EP0 and EP1 worker */
#define QUEUE_SIZE 32
struct buffer_job {
    uint32_t x;
    uint32_t y;
    uint32_t width;
    uint32_t height;
    uint32_t length;
    uint8_t compression;
    uint32_t compressed_length;
};

static struct {
    pthread_mutex_t mutex;
    pthread_cond_t cond;
    pthread_cond_t ready_cond;
    struct buffer_job jobs[QUEUE_SIZE];
    int head;
    int tail;
    int count;
    bool worker_ready;
} g_queue;

static void queue_init(void) {
    pthread_mutex_init(&g_queue.mutex, NULL);
    pthread_cond_init(&g_queue.cond, NULL);
    pthread_cond_init(&g_queue.ready_cond, NULL);
    g_queue.head = 0;
    g_queue.tail = 0;
    g_queue.count = 0;
    g_queue.worker_ready = true;
}

static void queue_push(struct buffer_job job) {
    pthread_mutex_lock(&g_queue.mutex);
    g_queue.worker_ready = false;
    if (g_queue.count < QUEUE_SIZE) {
        g_queue.jobs[g_queue.tail] = job;
        g_queue.tail = (g_queue.tail + 1) % QUEUE_SIZE;
        g_queue.count++;
        pthread_cond_signal(&g_queue.cond);
    } else {
        fprintf(stderr, "[-] Queue overflow, dropping oldest buffer\n");
        g_queue.head = (g_queue.head + 1) % QUEUE_SIZE;
        g_queue.jobs[g_queue.tail] = job;
        g_queue.tail = (g_queue.tail + 1) % QUEUE_SIZE;
    }
    pthread_mutex_unlock(&g_queue.mutex);
}

static void queue_wait_worker_ready(void) {
    pthread_mutex_lock(&g_queue.mutex);
    while (!g_queue.worker_ready && g_running) {
        pthread_cond_wait(&g_queue.ready_cond, &g_queue.mutex);
    }
    pthread_mutex_unlock(&g_queue.mutex);
    /* Give bulk_worker_thread 150us to enter read(g_ep1_fd) -> musb_gadget_queue */
    usleep(150);
}

static bool queue_pop(struct buffer_job *job) {
    pthread_mutex_lock(&g_queue.mutex);
    while (g_queue.count == 0 && g_running) {
        pthread_cond_wait(&g_queue.cond, &g_queue.mutex);
    }
    if (!g_running) {
        pthread_mutex_unlock(&g_queue.mutex);
        return false;
    }
    *job = g_queue.jobs[g_queue.head];
    g_queue.head = (g_queue.head + 1) % QUEUE_SIZE;
    g_queue.count--;
    if (g_queue.count == 0) {
        g_queue.worker_ready = true;
        pthread_cond_broadcast(&g_queue.ready_cond);
    }
    pthread_mutex_unlock(&g_queue.mutex);
    return true;
}

/* Hardware Watchdog Feeder Thread */
static void *watchdog_thread(void *arg) {
    (void)arg;
    int wdt = open("/dev/watchdog", O_WRONLY);
    if (wdt >= 0) {
        printf("[+] Hardware watchdog acquired. Feeding every 2s...\n");
        while (1) {
            write(wdt, "1", 1);
            sleep(2);
        }
        close(wdt);
    }
    return NULL;
}

/* Full read helper */
static ssize_t read_all(int fd, void *buf, size_t count) {
    size_t total = 0;
    while (total < count && g_running) {
        ssize_t r = read(fd, (char *)buf + total, count - total);
        if (r < 0) {
            if (errno == EINTR || errno == EAGAIN) continue;
            return -1;
        }
        if (r == 0) break;
        total += r;
    }
    return total;
}

/* Fast LZ4 raw block decompressor (matches Linux kernel LZ4_compress_default) */
static int lz4_decompress_block(const uint8_t *src, uint8_t *dst, int src_len, int dst_cap) {
    const uint8_t *ip = src;
    const uint8_t * const iend = src + src_len;
    uint8_t *op = dst;
    uint8_t * const oend = dst + dst_cap;

    while (ip < iend) {
        unsigned int token = *ip++;
        unsigned int lit_len = token >> 4;
        if (lit_len == 15) {
            unsigned int s;
            do {
                if (ip >= iend) return -1;
                s = *ip++;
                lit_len += s;
            } while (s == 255);
        }
        if (op + lit_len > oend || ip + lit_len > iend) return -1;
        memcpy(op, ip, lit_len);
        ip += lit_len;
        op += lit_len;

        if (ip >= iend) break;

        if (ip + 2 > iend) return -1;
        unsigned int offset = (unsigned int)ip[0] | ((unsigned int)ip[1] << 8);
        ip += 2;
        if (offset == 0 || op - offset < dst) return -1;
        const uint8_t *match = op - offset;

        unsigned int match_len = (token & 0x0f) + 4;
        if (match_len == 19) {
            unsigned int s;
            do {
                if (ip >= iend) return -1;
                s = *ip++;
                match_len += s;
            } while (s == 255);
        }
        if (op + match_len > oend) return -1;

        if (offset >= match_len) {
            memcpy(op, match, match_len);
            op += match_len;
        } else {
            while (match_len--) {
                *op++ = *match++;
            }
        }
    }
    return (int)(op - dst);
}

#include <time.h>

static inline uint64_t now_us(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (uint64_t)ts.tv_sec * 1000000ULL + (uint64_t)ts.tv_nsec / 1000ULL;
}

/* Bulk Pixel Consumer Thread (EP1) */
static void *bulk_worker_thread(void *arg) {
    (void)arg;
    printf("[+] Bulk pixel worker thread started. Opening %s...\n", EP1_PATH);
    g_ep1_fd = open(EP1_PATH, O_RDWR);
    if (g_ep1_fd < 0) {
        perror("[-] Failed to open EP1");
        return NULL;
    }
    printf("[+] EP1 bulk endpoint ready (LZ4 + Unit-Stride Delta Blit accelerated)!\n");

    uint8_t *bulk_buf = malloc(FRAME_SIZE);
    uint8_t *comp_buf = malloc(FRAME_SIZE);
    uint16_t *prev_fb = malloc(FRAME_SIZE);
    if (!bulk_buf || !comp_buf || !prev_fb) {
        perror("[-] Failed to allocate buffers");
        return NULL;
    }

    uint64_t frames = 0;
    uint64_t sum_usb_us = 0;
    uint64_t sum_blit_us = 0;
    uint64_t sum_changed_px = 0;
    struct buffer_job job;

    while (g_running) {
        if (!queue_pop(&job)) break;

        if (job.length == 0) {
            /* Consume 0-length bulk packet if host ever sends 0x0 rect */
            read(g_ep1_fd, bulk_buf, 64);
            continue;
        }
        if (job.length > FRAME_SIZE) {
            fprintf(stderr, "[-] Buffer length %u exceeds FRAME_SIZE %u\n", job.length, FRAME_SIZE);
            continue;
        }

        uint64_t t0 = now_us();
        if ((job.compression & GUD_COMPRESSION_LZ4) && job.compressed_length > 0 && job.compressed_length <= FRAME_SIZE) {
            ssize_t bytes_read = read_all(g_ep1_fd, comp_buf, job.compressed_length);
            if (bytes_read != (ssize_t)job.compressed_length) {
                if (g_running) {
                    fprintf(stderr, "[-] Incomplete LZ4 bulk read: expected %u, got %zd\n", job.compressed_length, bytes_read);
                }
                continue;
            }
            int decomp_len = lz4_decompress_block(comp_buf, bulk_buf, (int)job.compressed_length, (int)job.length);
            if (decomp_len != (int)job.length) {
                fprintf(stderr, "[-] LZ4 decompress mismatch: expected %u, got %d\n", job.length, decomp_len);
                continue;
            }
        } else {
            ssize_t bytes_read = read_all(g_ep1_fd, bulk_buf, job.length);
            if (bytes_read != (ssize_t)job.length) {
                if (g_running) {
                    fprintf(stderr, "[-] Incomplete raw bulk read: expected %u, got %zd\n", job.length, bytes_read);
                }
                continue;
            }
        }
        uint64_t t1 = now_us();

        /*
         * Blit pixels to framebuffer with hardware 90-degree CW rotation + unit-stride cached RAM delta check.
         * prev_fb is indexed in unrotated row-major order: prev_fb[(base_y + ry) * TOTAL_PANEL_WIDTH + (base_x + rx)]
         * so both src_row[rx] and prev_row[rx] are 100% sequential in L1 cache!
         */
        const uint16_t *src = (const uint16_t *)bulk_buf;
        uint32_t w = job.width;
        uint32_t h = job.height;
        uint32_t base_x = job.x;
        uint32_t base_y = job.y;

        uint32_t max_rx = w;
        if (g_active_mode_width <= GUD_VIEWPORT_WIDTH && base_x + max_rx > GUD_VIEWPORT_WIDTH) {
            max_rx = (base_x < GUD_VIEWPORT_WIDTH) ? (GUD_VIEWPORT_WIDTH - base_x) : 0;
        }
        if (base_x + max_rx > TOTAL_PANEL_WIDTH) {
            max_rx = (base_x < TOTAL_PANEL_WIDTH) ? (TOTAL_PANEL_WIDTH - base_x) : 0;
        }
        uint32_t max_ry = h;
        if (base_y + max_ry > SCREEN_HEIGHT) {
            max_ry = (base_y < SCREEN_HEIGHT) ? (SCREEN_HEIGHT - base_y) : 0;
        }

        bool full_repaint = g_force_full_frame;
        uint32_t changed_px = 0;
        for (uint32_t ry = 0; ry < max_ry; ry++) {
            uint32_t py = 599 - (base_y + ry);
            const uint16_t *src_row = &src[ry * w];
            uint16_t *prev_row = &prev_fb[(base_y + ry) * TOTAL_PANEL_WIDTH + base_x];
            uint32_t rx = 0;

            if (!full_repaint && !g_is_32bpp) {
                for (; rx + 4 <= max_rx; rx += 4) {
                    uint32_t s0 = *(const uint32_t *)&src_row[rx];
                    uint32_t p0 = *(const uint32_t *)&prev_row[rx];
                    uint32_t s1 = *(const uint32_t *)&src_row[rx + 2];
                    uint32_t p1 = *(const uint32_t *)&prev_row[rx + 2];
                    if (s0 == p0 && s1 == p1) continue;

                    *(uint32_t *)&prev_row[rx] = s0;
                    *(uint32_t *)&prev_row[rx + 2] = s1;

                    if ((uint16_t)s0 != (uint16_t)p0) {
                        g_fb_mem16[(base_x + rx) * 600 + py] = (uint16_t)s0;
                        changed_px++;
                    }
                    if ((uint16_t)(s0 >> 16) != (uint16_t)(p0 >> 16)) {
                        g_fb_mem16[(base_x + rx + 1) * 600 + py] = (uint16_t)(s0 >> 16);
                        changed_px++;
                    }
                    if ((uint16_t)s1 != (uint16_t)p1) {
                        g_fb_mem16[(base_x + rx + 2) * 600 + py] = (uint16_t)s1;
                        changed_px++;
                    }
                    if ((uint16_t)(s1 >> 16) != (uint16_t)(p1 >> 16)) {
                        g_fb_mem16[(base_x + rx + 3) * 600 + py] = (uint16_t)(s1 >> 16);
                        changed_px++;
                    }
                }
            }

            for (; rx < max_rx; rx++) {
                uint16_t c = src_row[rx];
                if (full_repaint || prev_row[rx] != c) {
                    prev_row[rx] = c;
                    uint32_t col_base = (base_x + rx) * 600;
                    if (g_is_32bpp) {
                        uint32_t r = ((c >> 11) & 0x1f) * 255 / 31;
                        uint32_t g = ((c >> 5) & 0x3f) * 255 / 63;
                        uint32_t b = (c & 0x1f) * 255 / 31;
                        g_fb_mem32[col_base + py] = 0xff000000 | (r << 16) | (g << 8) | b;
                    } else {
                        g_fb_mem16[col_base + py] = c;
                    }
                    changed_px++;
                }
            }
        }
        if (full_repaint) {
            g_force_full_frame = false;
        }
        uint64_t t2 = now_us();

        frames++;
        sum_usb_us += (t1 - t0);
        sum_blit_us += (t2 - t1);
        sum_changed_px += changed_px;
        if (frames % 30 == 0) {
            printf("[*] GUD: %llu frames | rect=%ux%u | avg usb=%.2fms blit=%.2fms changed=%llu px\n",
                   (unsigned long long)frames, w, h,
                   (double)sum_usb_us / 30000.0,
                   (double)sum_blit_us / 30000.0,
                   (unsigned long long)(sum_changed_px / 30));
            fflush(stdout);
            sum_usb_us = 0;
            sum_blit_us = 0;
            sum_changed_px = 0;
        }
    }

    free(prev_fb);
    free(comp_buf);
    free(bulk_buf);
    if (g_ep1_fd >= 0) close(g_ep1_fd);
    return NULL;
}

/* Write FunctionFS Descriptors and Strings to EP0 */
static int init_functionfs(void) {
    bool already_active = (access(EP1_PATH, F_OK) == 0);
    g_ep0_fd = open(EP0_PATH, O_RDWR);
    if (g_ep0_fd < 0) {
        perror("[-] Failed to open " EP0_PATH);
        return -1;
    }
    if (already_active) {
        printf("[+] FunctionFS %s already active — seamless zero-disconnect hot takeover!\n", EP1_PATH);
        return 0;
    }

    struct desc_v2 desc;
    memset(&desc, 0, sizeof(desc));
    desc.header.magic = htole32(FUNCTIONFS_DESCRIPTORS_MAGIC_V2);
    desc.header.flags = htole32(FUNCTIONFS_HAS_FS_DESC | FUNCTIONFS_HAS_HS_DESC);
    desc.header.length = htole32(sizeof(desc));
    desc.fs_count = htole32(2);
    desc.hs_count = htole32(2);

    /* Full-Speed Interface */
    desc.fs_descs.intf.bLength = sizeof(desc.fs_descs.intf);
    desc.fs_descs.intf.bDescriptorType = USB_DT_INTERFACE;
    desc.fs_descs.intf.bInterfaceNumber = 0;
    desc.fs_descs.intf.bAlternateSetting = 0;
    desc.fs_descs.intf.bNumEndpoints = 1;
    desc.fs_descs.intf.bInterfaceClass = USB_CLASS_VENDOR_SPEC;
    desc.fs_descs.intf.bInterfaceSubClass = 0;
    desc.fs_descs.intf.bInterfaceProtocol = 0;
    desc.fs_descs.intf.iInterface = 1;

    /* Full-Speed Bulk OUT Endpoint */
    desc.fs_descs.bulk_out.bLength = sizeof(desc.fs_descs.bulk_out);
    desc.fs_descs.bulk_out.bDescriptorType = USB_DT_ENDPOINT;
    desc.fs_descs.bulk_out.bEndpointAddress = 1 | USB_DIR_OUT;
    desc.fs_descs.bulk_out.bmAttributes = USB_ENDPOINT_XFER_BULK;
    desc.fs_descs.bulk_out.wMaxPacketSize = htole16(64);
    desc.fs_descs.bulk_out.bInterval = 0;

    /* High-Speed Interface & Endpoint */
    desc.hs_descs.intf = desc.fs_descs.intf;
    desc.hs_descs.bulk_out = desc.fs_descs.bulk_out;
    desc.hs_descs.bulk_out.wMaxPacketSize = htole16(512);

    ssize_t ret = write(g_ep0_fd, &desc, sizeof(desc));
    if (ret != sizeof(desc)) {
        perror("[-] Failed to write descriptors to ep0");
        close(g_ep0_fd);
        return -1;
    }

    struct strings str;
    memset(&str, 0, sizeof(str));
    str.header.magic = htole32(FUNCTIONFS_STRINGS_MAGIC);
    str.header.length = htole32(sizeof(str));
    str.header.str_count = htole32(1);
    str.header.lang_count = htole32(1);
    str.tab.code = htole16(0x0409);
    memcpy(str.tab.str1, "GUD USB Display\0", sizeof("GUD USB Display"));

    ret = write(g_ep0_fd, &str, sizeof(str));
    if (ret != sizeof(str)) {
        perror("[-] Failed to write strings to ep0");
        close(g_ep0_fd);
        return -1;
    }

    printf("[+] GUD FunctionFS descriptors & strings successfully written to ep0!\n");
    return 0;
}

/* Framebuffer Initialization */
static int init_framebuffer(void) {
    for (int retry = 0; retry < 15; retry++) {
        g_fb_fd = open(FB_PATH, O_RDWR);
        if (g_fb_fd >= 0) break;
        printf("[*] Waiting for %s to become available (%d/15)...\n", FB_PATH, retry + 1);
        sleep(1);
    }
    if (g_fb_fd < 0) {
        perror("[-] Failed to open " FB_PATH);
        return -1;
    }

    if (ioctl(g_fb_fd, FBIOGET_VSCREENINFO, &g_vinfo) < 0) {
        perror("[-] FBIOGET_VSCREENINFO failed");
        close(g_fb_fd);
        return -1;
    }

    g_vinfo.activate = FB_ACTIVATE_NOW | FB_ACTIVATE_FORCE;
    g_vinfo.xoffset = 0;
    g_vinfo.yoffset = 0;

    g_is_32bpp = (g_vinfo.bits_per_pixel == 32);
    g_fb_map_size = (size_t)TOTAL_PANEL_WIDTH * SCREEN_HEIGHT * (g_is_32bpp ? 4 : 2);

    void *mem = mmap(NULL, g_fb_map_size, PROT_READ | PROT_WRITE, MAP_SHARED, g_fb_fd, 0);
    if (mem == MAP_FAILED) {
        perror("[-] Framebuffer mmap failed");
        close(g_fb_fd);
        return -1;
    }

    if (g_is_32bpp) {
        g_fb_mem32 = (uint32_t *)mem;
    } else {
        g_fb_mem16 = (uint16_t *)mem;
    }

    printf("[+] Framebuffer %s mapped (%zu bytes, %ux%u @ %ubpp)\n",
           FB_PATH, g_fb_map_size, g_vinfo.xres, g_vinfo.yres, g_vinfo.bits_per_pixel);
    return 0;
}

/* Signal handler for clean termination */
static void sig_handler(int sig) {
    (void)sig;
    g_running = false;
    pthread_cond_broadcast(&g_queue.cond);
}

int main(int argc, char **argv) {
    (void)argc;
    (void)argv;
    setvbuf(stdout, NULL, _IOLBF, 0);
    setvbuf(stderr, NULL, _IONBF, 0);
    signal(SIGINT, sig_handler);
    signal(SIGTERM, sig_handler);

    printf("========================================================\n");
    printf("[*] Starting V553 GUD USB Display Hardware Emulator\n");
    printf("[*] 启动全志 V553 硬件级 USB 显示卡仿真守护进程 (GUD Native)\n");
    printf("========================================================\n");

    queue_init();

    if (init_framebuffer() < 0) {
        return 1;
    }

    pthread_t wdt_tid;
    pthread_create(&wdt_tid, NULL, watchdog_thread, NULL);

    if (init_functionfs() < 0) {
        return 1;
    }

    pthread_t worker_tid;
    pthread_create(&worker_tid, NULL, bulk_worker_thread, NULL);

    /* EP0 Control Event Loop */
    printf("[+] Entering EP0 control event processing loop...\n");
    struct usb_functionfs_event event;
    struct buffer_job pending_job;
    memset(&pending_job, 0, sizeof(pending_job));

    while (g_running) {
        ssize_t ret = read(g_ep0_fd, &event, sizeof(event));
        if (ret < 0) {
            if (errno == EINTR || errno == EAGAIN) continue;
            usleep(100000);
            continue;
        }

        switch (event.type) {
        case FUNCTIONFS_BIND:
            printf("[+] USB Gadget BIND event\n");
            break;
        case FUNCTIONFS_UNBIND:
            printf("[-] USB Gadget UNBIND event\n");
            break;
        case FUNCTIONFS_ENABLE:
            printf("[+] USB Gadget ENABLE event (Host configured device)\n");
            g_force_full_frame = true;
            break;
        case FUNCTIONFS_DISABLE:
            printf("[-] USB Gadget DISABLE event\n");
            break;
        case FUNCTIONFS_SUSPEND:
            printf("[*] USB Gadget SUSPEND event\n");
            break;
        case FUNCTIONFS_RESUME:
            printf("[*] USB Gadget RESUME event\n");
            break;
        case FUNCTIONFS_SETUP: {
            struct usb_ctrlrequest *setup = &event.u.setup;
            uint8_t req = setup->bRequest;
            uint16_t val = le16toh(setup->wValue);
            uint16_t idx = le16toh(setup->wIndex);
            uint16_t len = le16toh(setup->wLength);
            bool is_in = (setup->bRequestType & USB_DIR_IN) != 0;

            char zlp_buf = 0;

            if (req == GUD_REQ_GET_DESCRIPTOR) {
                struct gud_display_descriptor_req ddesc;
                memset(&ddesc, 0, sizeof(ddesc));
                ddesc.magic = htole32(GUD_DISPLAY_MAGIC);
                ddesc.version = 1;
                ddesc.flags = htole32(GUD_DISPLAY_FLAG_STATUS_ON_SET);
                ddesc.compression = GUD_COMPRESSION_LZ4;
                ddesc.max_buffer_size = htole32(FRAME_SIZE);
                ddesc.min_width = htole32(GUD_VIEWPORT_WIDTH);
                ddesc.max_width = htole32(TOTAL_PANEL_WIDTH);
                ddesc.min_height = htole32(SCREEN_HEIGHT);
                ddesc.max_height = htole32(SCREEN_HEIGHT);

                size_t send_len = len < sizeof(ddesc) ? len : sizeof(ddesc);
                write(g_ep0_fd, &ddesc, send_len);
            }
            else if (req == GUD_REQ_GET_FORMATS) {
                uint8_t fmt = GUD_PIXEL_FORMAT_RGB565;
                size_t send_len = len < sizeof(fmt) ? len : sizeof(fmt);
                write(g_ep0_fd, &fmt, send_len);
            }
            else if (req == GUD_REQ_GET_PROPERTIES) {
                /* No extra properties, send ZLP with valid user buffer */
                write(g_ep0_fd, &zlp_buf, 0);
            }
            else if (req == GUD_REQ_GET_CONNECTORS) {
                struct gud_connector_descriptor_req conn;
                memset(&conn, 0, sizeof(conn));
                conn.connector_type = GUD_CONNECTOR_TYPE_PANEL;
                conn.flags = 0;

                size_t send_len = len < sizeof(conn) ? len : sizeof(conn);
                write(g_ep0_fd, &conn, send_len);
            }
            else if (req == GUD_REQ_GET_CONNECTOR_PROPERTIES) {
                /* No connector properties, send ZLP */
                write(g_ep0_fd, &zlp_buf, 0);
            }
            else if (req == GUD_REQ_GET_CONNECTOR_STATUS) {
                uint8_t status = GUD_CONNECTOR_STATUS_CONNECTED;
                size_t send_len = len < sizeof(status) ? len : sizeof(status);
                write(g_ep0_fd, &status, send_len);
            }
            else if (req == GUD_REQ_GET_CONNECTOR_EDID) {
                /* No raw EDID, send ZLP so driver queries GET_CONNECTOR_MODES */
                write(g_ep0_fd, &zlp_buf, 0);
            }
            else if (req == GUD_REQ_GET_CONNECTOR_MODES) {
                struct gud_display_mode_req modes[2];
                memset(modes, 0, sizeof(modes));

                /* Mode 0 (Preferred): 1200x600 @ 60Hz - Dual-Zone Host Viewport */
                modes[0].clock = htole32(45826); /* 45.8 MHz */
                modes[0].hdisplay = htole16(GUD_VIEWPORT_WIDTH);
                modes[0].hsync_start = htole16(1216);
                modes[0].hsync_end = htole16(1232);
                modes[0].htotal = htole16(1248);
                modes[0].vdisplay = htole16(SCREEN_HEIGHT);
                modes[0].vsync_start = htole16(604);
                modes[0].vsync_end = htole16(608);
                modes[0].vtotal = htole16(612);
                modes[0].flags = htole32(GUD_DISPLAY_MODE_FLAG_PREFERRED |
                                         GUD_DISPLAY_MODE_FLAG_PHSYNC |
                                         GUD_DISPLAY_MODE_FLAG_PVSYNC);

                /* Mode 1 (Alternate): 1600x600 @ 60Hz - Full Panel Mode */
                modes[1].clock = htole32(57600); /* 57.6 MHz */
                modes[1].hdisplay = htole16(TOTAL_PANEL_WIDTH);
                modes[1].hsync_start = htole16(1616);
                modes[1].hsync_end = htole16(1632);
                modes[1].htotal = htole16(1648);
                modes[1].vdisplay = htole16(SCREEN_HEIGHT);
                modes[1].vsync_start = htole16(604);
                modes[1].vsync_end = htole16(608);
                modes[1].vtotal = htole16(612);
                modes[1].flags = htole32(GUD_DISPLAY_MODE_FLAG_PHSYNC |
                                         GUD_DISPLAY_MODE_FLAG_PVSYNC);

                size_t send_len = len < sizeof(modes) ? len : sizeof(modes);
                write(g_ep0_fd, modes, send_len);
            }
            else if (req == GUD_REQ_SET_BUFFER) {
                struct gud_set_buffer_req sbuf;
                if (len == sizeof(sbuf)) {
                    read(g_ep0_fd, &sbuf, sizeof(sbuf));
                    pending_job.x = le32toh(sbuf.x);
                    pending_job.y = le32toh(sbuf.y);
                    pending_job.width = le32toh(sbuf.width);
                    pending_job.height = le32toh(sbuf.height);
                    pending_job.length = le32toh(sbuf.length);
                    pending_job.compression = sbuf.compression;
                    pending_job.compressed_length = le32toh(sbuf.compressed_length);
                    if (pending_job.width > GUD_VIEWPORT_WIDTH || (pending_job.x + pending_job.width) > GUD_VIEWPORT_WIDTH) {
                        g_active_mode_width = TOTAL_PANEL_WIDTH;
                    }
                    queue_push(pending_job);
                } else if (len > 0) {
                    char dummy[128];
                    read(g_ep0_fd, dummy, len < sizeof(dummy) ? len : sizeof(dummy));
                }
            }
            else if (req == GUD_REQ_SET_STATE_CHECK) {
                if (len > 0) {
                    char dummy[128];
                    read(g_ep0_fd, dummy, len < sizeof(dummy) ? len : sizeof(dummy));
                }
            }
            else if (req == GUD_REQ_SET_STATE_COMMIT) {
                /* State committed, no payload */
                g_force_full_frame = true;
            }
            else if (req == GUD_REQ_SET_CONTROLLER_ENABLE || req == GUD_REQ_SET_DISPLAY_ENABLE) {
                if (len > 0) {
                    uint8_t en = 0;
                    read(g_ep0_fd, &en, 1);
                }
                g_force_full_frame = true;
            }
            else if (req == GUD_REQ_GET_STATUS) {
                queue_wait_worker_ready();
                uint8_t status = GUD_STATUS_OK;
                write(g_ep0_fd, &status, sizeof(status));
            }
            else {
                /* Unknown request: stall or consume */
                if (is_in) {
                    char dummy[8];
                    read(g_ep0_fd, dummy, sizeof(dummy));
                } else if (len > 0) {
                    char dummy[128];
                    read(g_ep0_fd, dummy, len < sizeof(dummy) ? len : sizeof(dummy));
                }
            }
            break;
        }
        default:
            break;
        }
    }

    printf("\n[*] Stopping GUD daemon...\n");
    g_running = false;
    pthread_cond_broadcast(&g_queue.cond);
    pthread_join(worker_tid, NULL);

    if (g_fb_mem32) munmap(g_fb_mem32, g_fb_map_size);
    else if (g_fb_mem16) munmap(g_fb_mem16, g_fb_map_size);
    if (g_fb_fd >= 0) close(g_fb_fd);
    if (g_ep0_fd >= 0) close(g_ep0_fd);

    printf("[+] Clean exit completed.\n");
    return 0;
}
