// SPDX-License-Identifier: GPL-2.0-only
/*
 * BOE 10.26-inch 600x1600 (1600x600 ultra-wide) MIPI-DSI Panel Driver (wtl1026boe2l)
 * Reverse-Engineered from Allwinner V553 factory firmware kernel
 *
 * Copyright (C) 2026 Allwinner & Antigravity
 */

#include <linux/delay.h>
#include <linux/gpio/consumer.h>
#include <linux/module.h>
#include <linux/of.h>

#include <drm/drm_mipi_dsi.h>
#include <drm/drm_modes.h>
#include <drm/drm_panel.h>
#include <drm/drm_probe_helper.h>

struct wtl1026boe2l_panel {
	struct drm_panel panel;
	struct mipi_dsi_device *dsi;
	struct gpio_desc *reset_gpio;
	struct gpio_desc *enable_gpio;
};

static inline struct wtl1026boe2l_panel *to_wtl1026boe2l_panel(struct drm_panel *panel)
{
	return container_of(panel, struct wtl1026boe2l_panel, panel);
}

static const struct drm_display_mode wtl1026boe2l_default_mode = {
	.clock = 53000,
	.hdisplay = 600,
	.hsync_start = 600 + 20,
	.hsync_end = 600 + 20 + 8,
	.htotal = 648,
	.vdisplay = 1600,
	.vsync_start = 1600 + 16,
	.vsync_end = 1600 + 16 + 2,
	.vtotal = 1634,
	.width_mm = 62,
	.height_mm = 165,
	.type = DRM_MODE_TYPE_DRIVER | DRM_MODE_TYPE_PREFERRED,
};

static int wtl1026boe2l_init_sequence(struct mipi_dsi_device *dsi)
{
	static const u8 cmd_b9[] = { 0xf1, 0x12, 0x87 };
	static const u8 cmd_b2[] = { 0x90, 0x05, 0x78 };
	static const u8 cmd_b3[] = { 0x10, 0x10, 0x28, 0x28, 0x03, 0xff, 0x00, 0x00, 0x00, 0x00 };
	static const u8 cmd_b4[] = { 0x80 };
	static const u8 cmd_b5[] = { 0x0c, 0x0c };
	static const u8 cmd_b6[] = { 0x6c, 0x6c };
	static const u8 cmd_b8[] = { 0x26, 0x22, 0xf0, 0x13 };
	static const u8 cmd_ba[] = { 0x31, 0x81, 0x05, 0xf9, 0x0e, 0x0e, 0x20, 0x00, 0x00, 0x00,
				     0x00, 0x00, 0x00, 0x00, 0x44, 0x25, 0x00, 0x91, 0x0a, 0x00,
				     0x00, 0x01, 0x4f, 0x01, 0x00, 0x00, 0x37 };
	static const u8 cmd_bc[] = { 0x47 };
	static const u8 cmd_bf[] = { 0x02, 0x10, 0x00, 0x80, 0x04 };
	static const u8 cmd_c0[] = { 0x73, 0x73, 0x50, 0x50, 0x00, 0x00, 0x12, 0x73, 0x00 };
	static const u8 cmd_c1[] = { 0x65, 0xc0, 0x32, 0x32, 0x77, 0xf4, 0x77, 0x77, 0xcc, 0xcc,
				     0xff, 0xff, 0x11, 0x11, 0x00, 0x00, 0x32 };
	static const u8 cmd_c7[] = { 0x10, 0x00, 0x0a, 0x00, 0x00, 0x00, 0x00, 0x00, 0xed, 0xc7, 0x00, 0xa5 };
	static const u8 cmd_c8[] = { 0x10, 0x40, 0x1e, 0x03 };
	static const u8 cmd_cc[] = { 0x0b };
	static const u8 cmd_e0[] = { 0x00, 0x0b, 0x11, 0x2d, 0x31, 0x38, 0x48, 0x40, 0x07, 0x0c,
				     0x0d, 0x12, 0x15, 0x11, 0x13, 0x11, 0x19, 0x00, 0x0b, 0x11,
				     0x2d, 0x31, 0x38, 0x48, 0x40, 0x07, 0x0c, 0x0d, 0x12, 0x15,
				     0x11, 0x13, 0x11, 0x19 };
	static const u8 cmd_e1[] = { 0x11, 0x11, 0x91, 0x00, 0x00, 0x00, 0x00 };
	static const u8 cmd_e3[] = { 0x07, 0x07, 0x0b, 0x0b, 0x0b, 0x0b, 0x00, 0x00, 0x00, 0x00,
				     0xff, 0x84, 0xc0, 0x10 };
	static const u8 cmd_e9[] = { 0xc8, 0x10, 0x07, 0x10, 0x0b, 0x80, 0x38, 0x12, 0x31, 0x23,
				     0x4f, 0x86, 0x80, 0x38, 0x47, 0x08, 0x00, 0xc0, 0x04, 0x00,
				     0x00, 0x00, 0x00, 0xc0, 0x04, 0x00, 0x00, 0x00, 0xf8, 0x48,
				     0xf8, 0x11, 0x33, 0x55, 0x77, 0x31, 0x88, 0x88, 0x88, 0xf8,
				     0x48, 0xf8, 0x00, 0x22, 0x44, 0x66, 0x20, 0x88, 0x88, 0x88,
				     0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00 };
	static const u8 cmd_ea[] = { 0x97, 0x12, 0x00, 0x00, 0x00, 0x78, 0x00, 0x00, 0x00, 0x00,
				     0x00, 0x00, 0x8f, 0x48, 0xf8, 0x66, 0x44, 0x22, 0x00, 0x02,
				     0x88, 0x88, 0x88, 0x8f, 0x48, 0xf8, 0x77, 0x55, 0x33, 0x11,
				     0x13, 0x88, 0x88, 0x88, 0x23, 0x00, 0x00, 0x01, 0x10, 0x00,
				     0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				     0x00, 0x00, 0x00, 0x00, 0x40, 0x80, 0x38, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00 };
	static const u8 cmd_ef[] = { 0xff, 0xff, 0x01 };

	int ret;

	dsi->mode_flags |= MIPI_DSI_MODE_LPM;

	ret = mipi_dsi_dcs_write(dsi, 0xb9, cmd_b9, sizeof(cmd_b9));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xb2, cmd_b2, sizeof(cmd_b2));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xb3, cmd_b3, sizeof(cmd_b3));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xb4, cmd_b4, sizeof(cmd_b4));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xb5, cmd_b5, sizeof(cmd_b5));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xb6, cmd_b6, sizeof(cmd_b6));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xb8, cmd_b8, sizeof(cmd_b8));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xba, cmd_ba, sizeof(cmd_ba));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xbc, cmd_bc, sizeof(cmd_bc));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xbf, cmd_bf, sizeof(cmd_bf));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xc0, cmd_c0, sizeof(cmd_c0));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xc1, cmd_c1, sizeof(cmd_c1));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xc7, cmd_c7, sizeof(cmd_c7));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xc8, cmd_c8, sizeof(cmd_c8));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xcc, cmd_cc, sizeof(cmd_cc));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xe0, cmd_e0, sizeof(cmd_e0));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xe1, cmd_e1, sizeof(cmd_e1));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xe3, cmd_e3, sizeof(cmd_e3));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xe9, cmd_e9, sizeof(cmd_e9));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xea, cmd_ea, sizeof(cmd_ea));
	if (ret < 0) return ret;
	ret = mipi_dsi_dcs_write(dsi, 0xef, cmd_ef, sizeof(cmd_ef));
	if (ret < 0) return ret;

	ret = mipi_dsi_dcs_exit_sleep_mode(dsi);
	if (ret < 0) return ret;
	msleep(250);

	ret = mipi_dsi_dcs_set_display_on(dsi);
	if (ret < 0) return ret;
	msleep(50);

	return 0;
}

static int wtl1026boe2l_prepare(struct drm_panel *panel)
{
	struct wtl1026boe2l_panel *ctx = to_wtl1026boe2l_panel(panel);
	int ret;

	if (ctx->enable_gpio)
		gpiod_set_value_cansleep(ctx->enable_gpio, 1);

	if (ctx->reset_gpio) {
		gpiod_set_value_cansleep(ctx->reset_gpio, 1);
		msleep(100);
		gpiod_set_value_cansleep(ctx->reset_gpio, 0);
		msleep(100);
		gpiod_set_value_cansleep(ctx->reset_gpio, 1);
		msleep(120);
	}

	ret = wtl1026boe2l_init_sequence(ctx->dsi);
	if (ret < 0) {
		dev_err(panel->dev, "Failed to send DSI init sequence: %d\n", ret);
		return ret;
	}

	return 0;
}

static int wtl1026boe2l_unprepare(struct drm_panel *panel)
{
	struct wtl1026boe2l_panel *ctx = to_wtl1026boe2l_panel(panel);

	mipi_dsi_dcs_set_display_off(ctx->dsi);
	msleep(50);
	mipi_dsi_dcs_enter_sleep_mode(ctx->dsi);
	msleep(100);

	if (ctx->reset_gpio)
		gpiod_set_value_cansleep(ctx->reset_gpio, 0);

	if (ctx->enable_gpio)
		gpiod_set_value_cansleep(ctx->enable_gpio, 0);

	return 0;
}

static int wtl1026boe2l_get_modes(struct drm_panel *panel,
				  struct drm_connector *connector)
{
	return drm_connector_helper_get_modes_fixed(connector, &wtl1026boe2l_default_mode);
}

static const struct drm_panel_funcs wtl1026boe2l_panel_funcs = {
	.prepare = wtl1026boe2l_prepare,
	.unprepare = wtl1026boe2l_unprepare,
	.get_modes = wtl1026boe2l_get_modes,
};

static int wtl1026boe2l_probe(struct mipi_dsi_device *dsi)
{
	struct device *dev = &dsi->dev;
	struct wtl1026boe2l_panel *ctx;
	int ret;

	ctx = devm_kzalloc(dev, sizeof(*ctx), GFP_KERNEL);
	if (!ctx)
		return -ENOMEM;

	ctx->reset_gpio = devm_gpiod_get_optional(dev, "reset", GPIOD_OUT_LOW);
	if (IS_ERR(ctx->reset_gpio))
		return dev_err_probe(dev, PTR_ERR(ctx->reset_gpio), "Failed to get reset-gpios\n");

	ctx->enable_gpio = devm_gpiod_get_optional(dev, "enable", GPIOD_OUT_LOW);
	if (IS_ERR(ctx->enable_gpio))
		return dev_err_probe(dev, PTR_ERR(ctx->enable_gpio), "Failed to get enable-gpios\n");

	ctx->dsi = dsi;
	mipi_dsi_set_drvdata(dsi, ctx);

	dsi->lanes = 2;
	dsi->format = MIPI_DSI_FMT_RGB888;
	dsi->mode_flags = MIPI_DSI_MODE_VIDEO | MIPI_DSI_MODE_VIDEO_BURST |
			  MIPI_DSI_MODE_LPM;

	drm_panel_init(&ctx->panel, dev, &wtl1026boe2l_panel_funcs,
		       DRM_MODE_CONNECTOR_DSI);
	ctx->panel.prepare_prev_first = true;

	ret = drm_panel_of_backlight(&ctx->panel);
	if (ret)
		return dev_err_probe(dev, ret, "Failed to get backlight\n");

	drm_panel_add(&ctx->panel);

	ret = mipi_dsi_attach(dsi);
	if (ret < 0) {
		drm_panel_remove(&ctx->panel);
		return dev_err_probe(dev, ret, "Failed to attach to DSI host\n");
	}

	dev_info(dev, "BOE wtl1026boe2l 600x1600 panel registered successfully\n");
	return 0;
}

static void wtl1026boe2l_remove(struct mipi_dsi_device *dsi)
{
	struct wtl1026boe2l_panel *ctx = mipi_dsi_get_drvdata(dsi);

	mipi_dsi_detach(dsi);
	drm_panel_remove(&ctx->panel);
}

static const struct of_device_id wtl1026boe2l_of_match[] = {
	{ .compatible = "boe,wtl1026boe2l" },
	{ /* sentinel */ }
};
MODULE_DEVICE_TABLE(of, wtl1026boe2l_of_match);

static struct mipi_dsi_driver wtl1026boe2l_driver = {
	.probe = wtl1026boe2l_probe,
	.remove = wtl1026boe2l_remove,
	.driver = {
		.name = "panel-boe-wtl1026boe2l",
		.of_match_table = wtl1026boe2l_of_match,
	},
};
module_mipi_dsi_driver(wtl1026boe2l_driver);

MODULE_AUTHOR("wen.chen <chenwen.beijing@gmail.com>");
MODULE_DESCRIPTION("DRM Panel Driver for BOE wtl1026boe2l 10.26-inch ultra-wide bar display");
MODULE_LICENSE("GPL");
