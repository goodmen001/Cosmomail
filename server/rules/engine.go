// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

// Package rules —— 邮件归档规则引擎：
// 新邮件同步入库前按账号加载启用的规则，按优先级依次匹配，
// 命中即返回目标目录 slug（mails.folder），未命中回落 inbox。
// 仅本地归档，不动服务器端邮件。
package rules

import (
	"strings"

	"novamail/models"

	"gorm.io/gorm"
)

// Engine 规则引擎
type Engine struct {
	db *gorm.DB
}

// NewEngine 创建规则引擎
func NewEngine(db *gorm.DB) *Engine { return &Engine{db: db} }

// MatchResult 匹配结果
type MatchResult struct {
	FolderSlug string // 命中的目录 slug；空串表示未命中（进 inbox）
	RuleID     uint   // 命中的规则 ID（0 表示未命中）
}

// Apply 对一封新邮件应用账号的归档规则，返回目标 folder。
// 规则按 priority 降序、id 升序排列，返回第一个命中的规则。
func (e *Engine) Apply(accountID uint, mail *models.Mail) MatchResult {
	var ruleList []models.MailRule
	if err := e.db.Where("account_id = ? AND enabled = ?", accountID, true).
		Order("priority DESC, id ASC").
		Find(&ruleList).Error; err != nil || len(ruleList) == 0 {
		return MatchResult{}
	}

	for _, r := range ruleList {
		if matchRule(&r, mail) {
			return MatchResult{FolderSlug: r.FolderSlug, RuleID: r.ID}
		}
	}
	return MatchResult{}
}

// matchRule 单条规则匹配（大小写不敏感）
func matchRule(r *models.MailRule, mail *models.Mail) bool {
	getField := func() string {
		switch r.Field {
		case "from":
			return mail.From
		case "subject":
			return mail.Subject
		case "to":
			return mail.To
		case "body":
			if mail.TextBody.Valid {
				return mail.TextBody.String
			}
			return ""
		default:
			return ""
		}
	}

	switch r.Operator {
	case "is_set": // 字段非空（主要用于 has_attachment）
		switch r.Field {
		case "has_attachment":
			return mail.HasAttachment
		default:
			return strings.TrimSpace(getField()) != ""
		}
	case "contains":
		return r.Value != "" && strings.Contains(strings.ToLower(getField()), strings.ToLower(r.Value))
	case "not_contains":
		return r.Value != "" && !strings.Contains(strings.ToLower(getField()), strings.ToLower(r.Value))
	case "equals":
		return r.Value != "" && strings.EqualFold(strings.TrimSpace(getField()), strings.TrimSpace(r.Value))
	case "starts_with":
		return r.Value != "" && strings.HasPrefix(strings.ToLower(getField()), strings.ToLower(r.Value))
	}
	return false
}
