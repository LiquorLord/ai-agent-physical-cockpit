#!/usr/bin/env python3
import sys

def patch_sun4i_tcon():
    path = "/root/mainline-linux/drivers/gpu/drm/sun4i/sun4i_tcon.c"
    with open(path, "r") as f:
        content = f.read()

    # 1. Update sun4i_tcon_enable_vblank to include SUN4I_TCON_GINT0_TCON0_TRI_COUNTER_ENABLE
    old_vblank = """	mask = SUN4I_TCON_GINT0_VBLANK_ENABLE(0) |
		SUN4I_TCON_GINT0_VBLANK_ENABLE(1) |
		SUN4I_TCON_GINT0_TCON0_TRI_FINISH_ENABLE;"""
    new_vblank = """	mask = SUN4I_TCON_GINT0_VBLANK_ENABLE(0) |
		SUN4I_TCON_GINT0_VBLANK_ENABLE(1) |
		SUN4I_TCON_GINT0_TCON0_TRI_FINISH_ENABLE |
		SUN4I_TCON_GINT0_TCON0_TRI_COUNTER_ENABLE;"""
    assert old_vblank in content, "Failed to find old_vblank in sun4i_tcon.c"
    content = content.replace(old_vblank, new_vblank)

    # 2. Update sun4i_tcon_handler to check and ack SUN4I_TCON_GINT0_TCON0_TRI_COUNTER_INT
    old_handler_check = """	if (!(status & (SUN4I_TCON_GINT0_VBLANK_INT(0) |
			SUN4I_TCON_GINT0_VBLANK_INT(1) |
			SUN4I_TCON_GINT0_TCON0_TRI_FINISH_INT)))
		return IRQ_NONE;"""
    new_handler_check = """	if (!(status & (SUN4I_TCON_GINT0_VBLANK_INT(0) |
			SUN4I_TCON_GINT0_VBLANK_INT(1) |
			SUN4I_TCON_GINT0_TCON0_TRI_FINISH_INT |
			SUN4I_TCON_GINT0_TCON0_TRI_COUNTER_INT)))
		return IRQ_NONE;"""
    assert old_handler_check in content, "Failed to find old_handler_check in sun4i_tcon.c"
    content = content.replace(old_handler_check, new_handler_check)

    old_handler_ack = """	regmap_update_bits(tcon->regs, SUN4I_TCON_GINT0_REG,
			   SUN4I_TCON_GINT0_VBLANK_INT(0) |
			   SUN4I_TCON_GINT0_VBLANK_INT(1) |
			   SUN4I_TCON_GINT0_TCON0_TRI_FINISH_INT,
			   0);"""
    new_handler_ack = """	regmap_update_bits(tcon->regs, SUN4I_TCON_GINT0_REG,
			   SUN4I_TCON_GINT0_VBLANK_INT(0) |
			   SUN4I_TCON_GINT0_VBLANK_INT(1) |
			   SUN4I_TCON_GINT0_TCON0_TRI_FINISH_INT |
			   SUN4I_TCON_GINT0_TCON0_TRI_COUNTER_INT,
			   0);"""
    assert old_handler_ack in content, "Failed to find old_handler_ack in sun4i_tcon.c"
    content = content.replace(old_handler_ack, new_handler_ack)

    # 3. Update sun4i_tcon0_mode_set_cpu with factory timing formulas
    old_cpu_tri = """	/*
	 * This looks suspicious, but it works...
	 *
	 * The datasheet says that this should be set higher than 20 *
	 * pixel cycle, but it's not clear what a pixel cycle is.
	 */
	regmap_read(tcon->regs, SUN4I_TCON0_DCLK_REG, &tcon_div);
	tcon_div &= GENMASK(6, 0);
	block_space = mode->htotal * bpp / (tcon_div * lanes);
	block_space -= mode->hdisplay + 40;

	regmap_write(tcon->regs, SUN4I_TCON0_CPU_TRI0_REG,
		     SUN4I_TCON0_CPU_TRI0_BLOCK_SPACE(block_space) |
		     SUN4I_TCON0_CPU_TRI0_BLOCK_SIZE(mode->hdisplay));

	regmap_write(tcon->regs, SUN4I_TCON0_CPU_TRI1_REG,
		     SUN4I_TCON0_CPU_TRI1_BLOCK_NUM(mode->vdisplay));

	start_delay = (mode->crtc_vtotal - mode->crtc_vdisplay - 10 - 1);
	start_delay = start_delay * mode->crtc_htotal * 149;
	start_delay = start_delay / (mode->crtc_clock / 1000) / 8;
	regmap_write(tcon->regs, SUN4I_TCON0_CPU_TRI2_REG,
		     SUN4I_TCON0_CPU_TRI2_TRANS_START_SET(10) |
		     SUN4I_TCON0_CPU_TRI2_START_DELAY(start_delay));

	/*
	 * The Allwinner BSP has a comment that the period should be
	 * the display clock * 15, but uses an hardcoded 3000...
	 */
	regmap_write(tcon->regs, SUN4I_TCON_SAFE_PERIOD_REG,
		     SUN4I_TCON_SAFE_PERIOD_NUM(3000) |
		     SUN4I_TCON_SAFE_PERIOD_MODE(3));"""

    new_cpu_tri = """	/*
	 * Factory Allwinner DSI timing reverse-engineered from sun8iw21 kernel:
	 * TRI0: space = htotal - hdisplay - 1 in high 12 bits, block size = hdisplay - 1 in low 12 bits
	 * TRI1: block num = vdisplay - 1
	 * TRI2: start delay in high 16 bits, burst transfer count = hdisplay - 1 in low 13 bits
	 * TRI3 (0x16c): auto-trigger counter and mode
	 */
	regmap_write(tcon->regs, SUN4I_TCON0_CPU_TRI0_REG,
		     (((mode->htotal - mode->hdisplay - 1) & 0xfff) << 16) |
		     ((mode->hdisplay - 1) & 0xfff));

	regmap_write(tcon->regs, SUN4I_TCON0_CPU_TRI1_REG,
		     SUN4I_TCON0_CPU_TRI1_BLOCK_NUM(mode->vdisplay));

	start_delay = (mode->crtc_vtotal - mode->crtc_vdisplay - 10 - 1);
	start_delay = start_delay * mode->crtc_htotal * 149;
	start_delay = start_delay / (mode->crtc_clock / 1000) / 8;
	regmap_write(tcon->regs, SUN4I_TCON0_CPU_TRI2_REG,
		     ((start_delay & 0xffff) << 16) |
		     ((mode->hdisplay - 1) & 0x1fff));

	{
		u32 tri_period = (mode->htotal * mode->vtotal) / 4;
		u32 tri_div = 1;
		while ((tri_period / tri_div) > 65534 && tri_div < 256)
			tri_div++;
		u32 tri_thresh = (tri_period / tri_div) - 1;
		regmap_write(tcon->regs, 0x16c,
			     0x10000000 | ((tri_thresh & 0xffff) << 8) | ((tri_div - 1) & 0xff));
	}

	/*
	 * Enable DCLK high nibble (0xF) for DSI sync/burst mode
	 */
	regmap_update_bits(tcon->regs, SUN4I_TCON0_DCLK_REG,
			   0xf0000000, 0xf0000000);

	/*
	 * The Allwinner BSP has a comment that the period should be
	 * the display clock * 15, but uses an hardcoded 3000...
	 */
	regmap_write(tcon->regs, SUN4I_TCON_SAFE_PERIOD_REG,
		     SUN4I_TCON_SAFE_PERIOD_NUM(3000) |
		     SUN4I_TCON_SAFE_PERIOD_MODE(3));"""

    assert old_cpu_tri in content, "Failed to find old_cpu_tri in sun4i_tcon.c"
    content = content.replace(old_cpu_tri, new_cpu_tri)

    with open(path, "w") as f:
        f.write(content)
    print("Patched sun4i_tcon.c successfully!")

def patch_sun6i_mipi_dsi():
    path = "/root/mainline-linux/drivers/gpu/drm/sun4i/sun6i_mipi_dsi.c"
    with open(path, "r") as f:
        content = f.read()

    old_enable = """	reset_control_deassert(dsi->reset);
	clk_prepare_enable(dsi->mod_clk);"""
    new_enable = """	reset_control_deassert(dsi->reset);
	clk_prepare_enable(dsi->bus_clk);
	clk_prepare_enable(dsi->mod_clk);"""
    assert old_enable in content, "Failed to find old_enable in sun6i_mipi_dsi.c"
    content = content.replace(old_enable, new_enable)

    old_disable = """	clk_disable_unprepare(dsi->mod_clk);
	reset_control_assert(dsi->reset);
	regulator_disable(dsi->regulator);"""
    new_disable = """	clk_disable_unprepare(dsi->mod_clk);
	clk_disable_unprepare(dsi->bus_clk);
	reset_control_assert(dsi->reset);
	regulator_disable(dsi->regulator);"""
    assert old_disable in content, "Failed to find old_disable in sun6i_mipi_dsi.c"
    content = content.replace(old_disable, new_disable)

    with open(path, "w") as f:
        f.write(content)
    print("Patched sun6i_mipi_dsi.c successfully!")

def patch_sun6i_dphy():
    path = "/root/mainline-linux/drivers/phy/allwinner/phy-sun6i-mipi-dphy.c"
    with open(path, "r") as f:
        content = f.read()

    # In probe: fetch dphy->bus_clk
    old_probe_clk = """	dphy->reset = devm_reset_control_get_shared(&pdev->dev, NULL);"""
    new_probe_clk = """	dphy->bus_clk = devm_clk_get(&pdev->dev, "bus");
	if (IS_ERR(dphy->bus_clk))
		return dev_err_probe(&pdev->dev, PTR_ERR(dphy->bus_clk), "Couldn't get DPHY bus clock\\n");

	dphy->reset = devm_reset_control_get_shared(&pdev->dev, NULL);"""
    assert old_probe_clk in content, "Failed to find old_probe_clk in phy-sun6i-mipi-dphy.c"
    content = content.replace(old_probe_clk, new_probe_clk)

    # In init: enable bus_clk
    old_init = """static int sun6i_dphy_init(struct phy *phy)
{
	struct sun6i_dphy *dphy = phy_get_drvdata(phy);

	reset_control_deassert(dphy->reset);
	clk_prepare_enable(dphy->mod_clk);
	clk_set_rate_exclusive(dphy->mod_clk, 150000000);

	return 0;
}"""
    new_init = """static int sun6i_dphy_init(struct phy *phy)
{
	struct sun6i_dphy *dphy = phy_get_drvdata(phy);

	reset_control_deassert(dphy->reset);
	clk_prepare_enable(dphy->bus_clk);
	clk_prepare_enable(dphy->mod_clk);
	clk_set_rate_exclusive(dphy->mod_clk, 150000000);

	return 0;
}"""
    assert old_init in content, "Failed to find old_init in phy-sun6i-mipi-dphy.c"
    content = content.replace(old_init, new_init)

    # In exit: disable bus_clk
    old_exit = """static int sun6i_dphy_exit(struct phy *phy)
{
	struct sun6i_dphy *dphy = phy_get_drvdata(phy);

	clk_rate_exclusive_put(dphy->mod_clk);
	clk_disable_unprepare(dphy->mod_clk);
	reset_control_assert(dphy->reset);

	return 0;
}"""
    new_exit = """static int sun6i_dphy_exit(struct phy *phy)
{
	struct sun6i_dphy *dphy = phy_get_drvdata(phy);

	clk_rate_exclusive_put(dphy->mod_clk);
	clk_disable_unprepare(dphy->mod_clk);
	clk_disable_unprepare(dphy->bus_clk);
	reset_control_assert(dphy->reset);

	return 0;
}"""
    assert old_exit in content, "Failed to find old_exit in phy-sun6i-mipi-dphy.c"
    content = content.replace(old_exit, new_exit)

    with open(path, "w") as f:
        f.write(content)
    print("Patched phy-sun6i-mipi-dphy.c successfully!")

if __name__ == "__main__":
    patch_sun4i_tcon()
    patch_sun6i_mipi_dsi()
    patch_sun6i_dphy()
