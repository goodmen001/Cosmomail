// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package handlers

import (
	"novamail/models"
	"novamail/services"

	"github.com/gofiber/fiber/v2"
)

// GotifyHandler Gotify 通知配置 Handler
type GotifyHandler struct {
	service *services.GotifyService
}

// NewGotifyHandler 创建 Gotify Handler
func NewGotifyHandler(svc *services.GotifyService) *GotifyHandler {
	return &GotifyHandler{service: svc}
}

// Get 获取当前用户的 Gotify 配置（token 打码）
func (h *GotifyHandler) Get(c *fiber.Ctx) error {
	cfg := h.service.GetConfig(currentUserID(c))
	masked := cfg.Token
	if len(masked) > 8 {
		masked = masked[:4] + "****" + masked[len(masked)-4:]
	} else if masked != "" {
		masked = "****"
	}
	return c.JSON(fiber.Map{"enabled": cfg.Enabled, "url": cfg.URL, "token": masked, "configured": cfg.Token != ""})
}

// Save 保存当前用户的 Gotify 配置
func (h *GotifyHandler) Save(c *fiber.Ctx) error {
	var req models.GotifyConfig
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "请求参数解析失败"})
	}
	// token 为打码值时保留原 token
	if req.Token != "" && (req.Token == "****" || (len(req.Token) > 9 && req.Token[4:8] == "****")) {
		req.Token = h.service.GetConfig(currentUserID(c)).Token
	}
	if err := h.service.SaveConfig(currentUserID(c), req); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "保存失败", "detail": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

// Test 发送测试通知
func (h *GotifyHandler) Test(c *fiber.Ctx) error {
	var req models.GotifyConfig
	_ = c.BodyParser(&req)
	if req.URL == "" || req.Token == "" || req.Token == "****" {
		// 用已保存配置测试
		saved := h.service.GetConfig(currentUserID(c))
		if req.URL != "" {
			saved.URL = req.URL
		}
		req = saved
	}
	if err := h.service.Test(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "测试失败: " + err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "测试消息已发送，请在 Gotify 客户端查看"})
}
