#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>
#include <endian.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <linux/types.h>
#include <linux/usb/ch9.h>
#include <linux/usb/functionfs.h>
#include <stdint.h>

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

int main(void) {
    printf("[*] Testing FunctionFS descriptors on /dev/ffs-gud/ep0...\n");
    int fd = open("/dev/ffs-gud/ep0", O_RDWR);
    if (fd < 0) {
        perror("[-] Failed to open /dev/ffs-gud/ep0");
        return 1;
    }

    struct desc_v2 desc;
    memset(&desc, 0, sizeof(desc));
    desc.header.magic = htole32(FUNCTIONFS_DESCRIPTORS_MAGIC_V2);
    desc.header.flags = htole32(FUNCTIONFS_HAS_FS_DESC | FUNCTIONFS_HAS_HS_DESC);
    desc.header.length = htole32(sizeof(desc));
    desc.fs_count = htole32(2);
    desc.hs_count = htole32(2);

    // Full Speed
    desc.fs_descs.intf.bLength = sizeof(desc.fs_descs.intf);
    desc.fs_descs.intf.bDescriptorType = USB_DT_INTERFACE;
    desc.fs_descs.intf.bInterfaceNumber = 0;
    desc.fs_descs.intf.bAlternateSetting = 0;
    desc.fs_descs.intf.bNumEndpoints = 1;
    desc.fs_descs.intf.bInterfaceClass = USB_CLASS_VENDOR_SPEC;
    desc.fs_descs.intf.bInterfaceSubClass = 0;
    desc.fs_descs.intf.bInterfaceProtocol = 0;
    desc.fs_descs.intf.iInterface = 1;

    desc.fs_descs.bulk_out.bLength = sizeof(desc.fs_descs.bulk_out);
    desc.fs_descs.bulk_out.bDescriptorType = USB_DT_ENDPOINT;
    desc.fs_descs.bulk_out.bEndpointAddress = 1 | USB_DIR_OUT;
    desc.fs_descs.bulk_out.bmAttributes = USB_ENDPOINT_XFER_BULK;
    desc.fs_descs.bulk_out.wMaxPacketSize = htole16(64);
    desc.fs_descs.bulk_out.bInterval = 0;

    // High Speed
    desc.hs_descs.intf = desc.fs_descs.intf;
    desc.hs_descs.bulk_out = desc.fs_descs.bulk_out;
    desc.hs_descs.bulk_out.wMaxPacketSize = htole16(512);

    ssize_t ret = write(fd, &desc, sizeof(desc));
    if (ret < 0) {
        perror("[-] Failed to write descriptors to ep0");
        close(fd);
        return 1;
    }
    printf("[+] Successfully wrote %zd bytes descriptors to ep0!\n", ret);

    struct strings str;
    memset(&str, 0, sizeof(str));
    str.header.magic = htole32(FUNCTIONFS_STRINGS_MAGIC);
    str.header.length = htole32(sizeof(str));
    str.header.str_count = htole32(1);
    str.header.lang_count = htole32(1);
    str.tab.code = htole16(0x0409);
    memcpy(str.tab.str1, "GUD USB Display\0", sizeof("GUD USB Display"));

    ret = write(fd, &str, sizeof(str));
    if (ret < 0) {
        perror("[-] Failed to write strings to ep0");
        close(fd);
        return 1;
    }
    printf("[+] Successfully wrote %zd bytes strings to ep0!\n", ret);

    printf("[*] Now reading events on ep0...\n");
    struct usb_functionfs_event event;
    while (1) {
        ret = read(fd, &event, sizeof(event));
        if (ret < 0) {
            perror("[-] Failed to read event from ep0");
            break;
        }
        printf("[*] Received event type=%d\n", event.type);
        if (event.type == FUNCTIONFS_SETUP) {
            printf("[*] SETUP: bRequestType=0x%02x bRequest=0x%02x wValue=0x%04x wIndex=0x%04x wLength=%u\n",
                   event.u.setup.bRequestType,
                   event.u.setup.bRequest,
                   le16toh(event.u.setup.wValue),
                   le16toh(event.u.setup.wIndex),
                   le16toh(event.u.setup.wLength));
            if (event.u.setup.bRequestType & USB_DIR_IN) {
                // Return stall or dummy byte for now
                char dummy[64] = {0};
                uint16_t wlen = le16toh(event.u.setup.wLength);
                uint16_t send_len = wlen < sizeof(dummy) ? wlen : sizeof(dummy);
                write(fd, dummy, send_len);
            } else if (le16toh(event.u.setup.wLength) > 0) {
                char dummy[64];
                read(fd, dummy, le16toh(event.u.setup.wLength));
            }
        }
    }

    close(fd);
    return 0;
}
