// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package handlers

import (
	"novamail/models"
	"novamail/services"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
)

// AuthHandler 认证相关 Handler
type AuthHandler struct {
	service *services.AuthService
}

// NewAuthHandler 创建认证 Handler
func NewAuthHandler(svc *services.AuthService) *AuthHandler {
	return &AuthHandler{service: svc}
}

// Login 登录
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "请求参数无效: " + err.Error()})
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "用户名和密码不能为空"})
	}
	if len(req.Password) > 72 {
		return c.Status(400).JSON(fiber.Map{"error": "密码不能超过 72 字节"})
	}

	result, err := h.service.Login(req)
	if err != nil {
		msg := "登录失败"
		if err == services.ErrInvalidCredentials {
			msg = "用户名或密码错误"
		}
		return c.Status(401).JSON(fiber.Map{"error": msg, "detail": err.Error()})
	}
	return c.JSON(result)
}

// Register 注册（首次初始化，单用户模式）
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req models.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "请求参数无效: " + err.Error()})
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "用户名和密码不能为空"})
	}
	usernameLength := utf8.RuneCountInString(req.Username)
	if usernameLength < 3 || usernameLength > 32 {
		return c.Status(400).JSON(fiber.Map{"error": "用户名长度应为 3–32 个字符"})
	}
	if len(req.Password) < 8 || len(req.Password) > 72 {
		return c.Status(400).JSON(fiber.Map{"error": "密码长度应为 8–72 字节"})
	}

	result, err := h.service.Register(req)
	if err != nil {
		statusCode := 500
		msg := "注册失败"
		if err == services.ErrUserExists {
			statusCode = 409
			msg = "用户名已被注册"
		} else if err == services.ErrRegistrationClosed {
			statusCode = 403
			msg = "当前未开放注册，请联系管理员"
		}
		return c.Status(statusCode).JSON(fiber.Map{"error": msg, "detail": err.Error()})
	}

	c.Status(201)
	return c.JSON(result)
}

// Status 查询认证状态（是否需要初始化）
func (h *AuthHandler) Status(c *fiber.Ctx) error {
	status := h.service.GetAuthStatus()
	return c.JSON(status)
}

// GetRegistrationEnabled 查询注册开关（公开）
func (h *AuthHandler) GetRegistrationEnabled(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"enabled": h.service.GetRegistrationEnabled()})
}

// SetRegistrationEnabled 管理员切换注册开关
func (h *AuthHandler) SetRegistrationEnabled(c *fiber.Ctx) error {
	var req models.RegisterEnabledRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "请求参数解析失败"})
	}
	if err := h.service.SetRegistrationEnabled(req.Enabled); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "设置失败", "detail": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "enabled": req.Enabled})
}

// ListUsers 用户列表（管理员）
func (h *AuthHandler) ListUsers(c *fiber.Ctx) error {
	users, err := h.service.ListUsers()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "查询失败", "detail": err.Error()})
	}
	return c.JSON(fiber.Map{"users": users})
}

// DeleteUser 删除用户（管理员）
func (h *AuthHandler) DeleteUser(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "无效的 ID"})
	}
	operatorID := currentUserID(c)
	if err := h.service.DeleteUser(operatorID, uint(id)); err != nil {
		status := 500
		if err == services.ErrForbidden {
			status = 403
		} else if err == services.ErrLastAdmin {
			status = 400
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "用户已删除"})
}

// Me 返回当前登录用户的实时信息（用户名、数据库中的管理员身份）
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	item, err := h.service.Me(currentUserID(c))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "用户不存在"})
	}
	return c.JSON(item)
}

// ChangePassword 修改当前用户密码
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	var req models.ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "请求参数解析失败"})
	}
	if len(req.NewPassword) < 8 || len(req.NewPassword) > 72 {
		return c.Status(400).JSON(fiber.Map{"error": "新密码长度应为 8–72 字节"})
	}
	if err := h.service.ChangePassword(currentUserID(c), req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "密码已修改"})
}

// CreateUser 管理员手动创建用户
func (h *AuthHandler) CreateUser(c *fiber.Ctx) error {
	var req models.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "请求参数解析失败"})
	}
	item, err := h.service.CreateUser(req)
	if err != nil {
		status := 400
		if err == services.ErrUserExists {
			status = 409
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(item)
}

// SetUserDisabled 管理员禁用/启用用户
func (h *AuthHandler) SetUserDisabled(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "无效的 ID"})
	}
	var req models.SetUserDisabledRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "请求参数解析失败"})
	}
	if err := h.service.SetUserDisabled(currentUserID(c), uint(id), req.Disabled); err != nil {
		status := 400
		if err == services.ErrForbidden || strings.Contains(err.Error(), "自己") {
			status = 403
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}

// GetApiToken 获取（或首次生成）当前用户 API Token
func (h *AuthHandler) GetApiToken(c *fiber.Ctx) error {
	tok, err := h.service.EnsureApiToken(currentUserID(c))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"token": tok})
}

// RevokeApiToken 吊销当前用户 API Token
func (h *AuthHandler) RevokeApiToken(c *fiber.Ctx) error {
	if err := h.service.RevokeApiToken(currentUserID(c)); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "吊销失败"})
	}
	return c.JSON(fiber.Map{"success": true})
}
