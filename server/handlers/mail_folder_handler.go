// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"novamail/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// MailFolderHandler 自定义邮件目录 + 归档规则 API 处理器
type MailFolderHandler struct {
	db *gorm.DB
}

// NewMailFolderHandler 创建 Handler
func NewMailFolderHandler(db *gorm.DB) *MailFolderHandler {
	return &MailFolderHandler{db: db}
}

// --- 目录 CRUD ---

// Folders 指定账号的目录列表（GET /mail-folders?account_id=N）
func (h *MailFolderHandler) Folders(c *fiber.Ctx) error {
	userID := currentUserID(c)
	accountID, err := strconv.Atoi(c.Query("account_id"))
	if err != nil || accountID <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "需要 account_id 参数"})
	}
	var folders []models.MailFolder
	if err := h.db.Where("user_id = ? AND account_id = ?", userID, accountID).
		Order("sort_order ASC, id ASC").Find(&folders).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "查询失败", "detail": err.Error()})
	}
	return c.JSON(fiber.Map{"folders": folders})
}

// CreateFolder 创建目录（POST /mail-folders）
func (h *MailFolderHandler) CreateFolder(c *fiber.Ctx) error {
	userID := currentUserID(c)
	var req models.MailFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "参数错误"})
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "目录名称不能为空"})
	}
	accountID, err := strconv.Atoi(c.Query("account_id"))
	if err != nil || accountID <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "需要 account_id 参数"})
	}
	if !h.ownsAccount(userID, uint(accountID)) {
		return c.Status(403).JSON(fiber.Map{"error": "无权访问该邮箱账号"})
	}

	folder := models.MailFolder{
		UserID:    userID,
		AccountID: uint(accountID),
		Name:      name,
		SortOrder: req.SortOrder,
	}
	// slug 生成 + 同账号内唯一
	base := slugify(name)
	for i := 0; i < 50; i++ {
		slug := base
		if i > 0 {
			slug = fmt.Sprintf("%s-%d", base, i)
		}
		if len(slug) > 64 {
			slug = slug[:64]
		}
		var n int64
		h.db.Model(&models.MailFolder{}).
			Where("account_id = ? AND slug = ?", accountID, slug).Count(&n)
		if n == 0 {
			folder.Slug = slug
			break
		}
	}
	if folder.Slug == "" {
		return c.Status(500).JSON(fiber.Map{"error": "无法生成唯一目录标识"})
	}
	if err := h.db.Create(&folder).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "创建失败", "detail": err.Error()})
	}
	return c.JSON(folder)
}

// UpdateFolder 重命名/排序（PUT /mail-folders/:id）
func (h *MailFolderHandler) UpdateFolder(c *fiber.Ctx) error {
	userID := currentUserID(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "无效 ID"})
	}
	var req models.MailFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "参数错误"})
	}
	var folder models.MailFolder
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&folder).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "目录不存在"})
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		folder.Name = name
	}
	folder.SortOrder = req.SortOrder
	if err := h.db.Save(&folder).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "保存失败", "detail": err.Error()})
	}
	return c.JSON(folder)
}

// DeleteFolder 删除目录（DELETE /mail-folders/:id）
// 目录下的邮件全部退回 inbox；引用该目录的规则一并删除。
func (h *MailFolderHandler) DeleteFolder(c *fiber.Ctx) error {
	userID := currentUserID(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "无效 ID"})
	}
	var folder models.MailFolder
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&folder).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "目录不存在"})
	}
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Mail{}).
			Where("account_id = ? AND folder = ?", folder.AccountID, folder.Slug).
			Update("folder", models.FolderInbox).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ? AND folder_slug = ?", folder.AccountID, folder.Slug).
			Delete(&models.MailRule{}).Error; err != nil {
			return err
		}
		return tx.Delete(&folder).Error
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "删除失败", "detail": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// --- 规则 CRUD ---

// Rules 指定账号的规则列表（GET /mail-rules?account_id=N）
func (h *MailFolderHandler) Rules(c *fiber.Ctx) error {
	userID := currentUserID(c)
	accountID, err := strconv.Atoi(c.Query("account_id"))
	if err != nil || accountID <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "需要 account_id 参数"})
	}
	var ruleList []models.MailRule
	if err := h.db.Where("user_id = ? AND account_id = ?", userID, accountID).
		Order("priority DESC, id ASC").Find(&ruleList).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "查询失败", "detail": err.Error()})
	}
	return c.JSON(fiber.Map{"rules": ruleList})
}

var validRuleFields = map[string]bool{
	"from": true, "subject": true, "to": true, "body": true, "has_attachment": true,
}
var validRuleOperators = map[string]bool{
	"contains": true, "not_contains": true, "equals": true, "starts_with": true, "is_set": true,
}

// CreateRule 创建规则（POST /mail-rules?account_id=N）
func (h *MailFolderHandler) CreateRule(c *fiber.Ctx) error {
	userID := currentUserID(c)
	accountID, err := strconv.Atoi(c.Query("account_id"))
	if err != nil || accountID <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "需要 account_id 参数"})
	}
	if !h.ownsAccount(userID, uint(accountID)) {
		return c.Status(403).JSON(fiber.Map{"error": "无权访问该邮箱账号"})
	}
	var req models.MailRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "参数错误"})
	}
	if err := h.validateRule(uint(accountID), userID, &req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rule := models.MailRule{
		UserID: userID, AccountID: uint(accountID),
		Name:  strings.TrimSpace(req.Name),
		Field: req.Field, Operator: req.Operator, Value: req.Value,
		FolderSlug: req.FolderSlug, Priority: req.Priority, Enabled: enabled,
	}
	if err := h.db.Create(&rule).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "创建失败", "detail": err.Error()})
	}
	return c.JSON(rule)
}

// UpdateRule 修改规则（PUT /mail-rules/:id）
func (h *MailFolderHandler) UpdateRule(c *fiber.Ctx) error {
	userID := currentUserID(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "无效 ID"})
	}
	var rule models.MailRule
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&rule).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "规则不存在"})
	}
	var req models.MailRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "参数错误"})
	}
	if err := h.validateRule(rule.AccountID, userID, &req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	rule.Name = strings.TrimSpace(req.Name)
	rule.Field = req.Field
	rule.Operator = req.Operator
	rule.Value = req.Value
	rule.FolderSlug = req.FolderSlug
	rule.Priority = req.Priority
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if err := h.db.Save(&rule).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "保存失败", "detail": err.Error()})
	}
	return c.JSON(rule)
}

// DeleteRule 删除规则（DELETE /mail-rules/:id）
func (h *MailFolderHandler) DeleteRule(c *fiber.Ctx) error {
	userID := currentUserID(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "无效 ID"})
	}
	res := h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.MailRule{})
	if res.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "删除失败", "detail": res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "规则不存在"})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// validateRule 校验规则字段/操作/目标目录
func (h *MailFolderHandler) validateRule(accountID, userID uint, req *models.MailRuleRequest) error {
	if !validRuleFields[req.Field] {
		return fmt.Errorf("无效的匹配字段")
	}
	if !validRuleOperators[req.Operator] {
		return fmt.Errorf("无效的匹配操作")
	}
	if req.Operator != "is_set" && strings.TrimSpace(req.Value) == "" {
		return fmt.Errorf("匹配值不能为空")
	}
	var n int64
	h.db.Model(&models.MailFolder{}).
		Where("account_id = ? AND user_id = ? AND slug = ?", accountID, userID, req.FolderSlug).Count(&n)
	if n == 0 {
		return fmt.Errorf("目标目录不存在")
	}
	return nil
}

// ownsAccount 校验账号归属当前用户
func (h *MailFolderHandler) ownsAccount(userID, accountID uint) bool {
	var n int64
	h.db.Model(&models.MailAccount{}).Where("id = ? AND user_id = ?", accountID, userID).Count(&n)
	return n > 0
}

// slugify 由名称生成目录 slug（小写字母数字连字符；非 ASCII 字符转为拼音式占位或截断）
func slugify(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == '.':
			b.WriteByte('-')
		default:
			// 中文等非 ASCII 字符统一省略，避免 slug 编码问题
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		s = "folder"
	}
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}
