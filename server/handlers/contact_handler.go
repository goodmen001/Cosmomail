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
	accountID := uint(0)
	if v := c.Query("account_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			accountID = uint(n)
		}
	}
	contacts, err := h.service.ListContacts(currentUserID(c), accountID, keyword, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "查询失败", "detail": err.Error()})
	}
	return c.JSON(fiber.Map{"contacts": contacts, "total": len(contacts)})
}

// Sync 全量提取指定账号已同步邮件中的联系人（POST /contacts/sync?account_id=N）
func (h *ContactHandler) Sync(c *fiber.Ctx) error {
	accountID, err := strconv.Atoi(c.Query("account_id"))
	if err != nil || accountID <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "需要 account_id 参数"})
	}
	n, err := h.service.SyncAccountContacts(currentUserID(c), uint(accountID))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "同步失败", "detail": err.Error()})
	}
	return c.JSON(fiber.Map{"extracted": n})
}

// SyncCardDAV 从指定账号配置的 CardDAV 服务器同步联系人（POST /contacts/carddav/sync?account_id=N）
func (h *ContactHandler) SyncCardDAV(c *fiber.Ctx) error {
	accountID, err := strconv.Atoi(c.Query("account_id"))
	if err != nil || accountID <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "需要 account_id 参数"})
	}
	n, err := h.service.SyncAccountCardDAV(currentUserID(c), uint(accountID))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "CardDAV 同步失败", "detail": err.Error()})
	}
	return c.JSON(fiber.Map{"extracted": n})
}
