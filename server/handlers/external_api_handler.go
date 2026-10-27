// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package handlers

import (
	"strconv"

	"novamail/models"
	"novamail/services"

	"github.com/gofiber/fiber/v2"
)

// ExternalApiHandler 面向外部 AI / 脚本的只读邮件 API（X-API-Token 认证，每用户独立）
type ExternalApiHandler struct {
	mailService *services.MailService
}

// NewExternalApiHandler 创建外部 API Handler
func NewExternalApiHandler(mailService *services.MailService) *ExternalApiHandler {
	return &ExternalApiHandler{mailService: mailService}
}

// apiUserID 从 ApiTokenAuth 中间件取用户 ID
func apiUserID(c *fiber.Ctx) uint {
	if id, ok := c.Locals("user_id").(uint); ok {
		return id
	}
	if f, ok := c.Locals("user_id").(float64); ok {
		return uint(f)
	}
	return 0
}

// ListMails GET /api/v1/external/mails?folder=inbox&page=1&page_size=20&keyword=xxx
func (h *ExternalApiHandler) ListMails(c *fiber.Ctx) error {
	filter := models.MailListFilter{
		Page:     c.QueryInt("page", 1),
		PageSize: c.QueryInt("page_size", 20),
		Folder:   c.Query("folder"),
		Keyword:  c.Query("keyword"),
	}
	if c.Query("is_read") != "" {
		v := c.Query("is_read") == "true"
		filter.IsRead = &v
	}
	if c.Query("account_id") != "" {
		if id, err := strconv.Atoi(c.Query("account_id")); err == nil {
			filter.AccountID = uint(id)
		}
	}

	items, total, err := h.mailService.List(apiUserID(c), filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "查询失败", "detail": err.Error()})
	}
	return c.JSON(fiber.Map{"total": total, "page": filter.Page, "page_size": filter.PageSize, "mails": items})
}

// GetMail GET /api/v1/external/mails/:id —— 返回邮件全文（含正文、附件元数据）
func (h *ExternalApiHandler) GetMail(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "无效的邮件 ID"})
	}
	mail, err := h.mailService.GetByID(apiUserID(c), uint(id))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "邮件不存在"})
	}
	return c.JSON(mail)
}

// GetStats GET /api/v1/external/mails/stats
func (h *ExternalApiHandler) GetStats(c *fiber.Ctx) error {
	return c.JSON(h.mailService.GetStats(apiUserID(c), nil))
}
