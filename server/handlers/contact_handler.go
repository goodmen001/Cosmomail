// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package handlers

import (
	"cosmomail/services"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// ContactHandler 联系人 API 处理器（发信收件人记录）
type ContactHandler struct {
	service *services.MailService
}

// NewContactHandler 创建联系人 Handler
func NewContactHandler(svc *services.MailService) *ContactHandler {
	return &ContactHandler{service: svc}
}

// List 联系人列表（当前用户，按最近使用排序，支持关键词搜索）
func (h *ContactHandler) List(c *fiber.Ctx) error {
	keyword := c.Query("keyword")
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	contacts, err := h.service.ListContacts(currentUserID(c), keyword, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "查询失败", "detail": err.Error()})
	}
	return c.JSON(fiber.Map{"contacts": contacts, "total": len(contacts)})
}
