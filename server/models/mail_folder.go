// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package models

import "time"

// 自定义目录所属的系统保留 folder 值（ mails.folder 字段既有取值）
const (
	FolderInbox = "inbox"
	FolderSent  = "sent"
)

// MailFolder 用户自定义邮件目录（按邮箱账号维度，每个账号一套）
// 目录创建后以 slug 形式写入 mails.folder，实现本地归档
type MailFolder struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index;not null;default:0;comment:所属用户ID(数据隔离)"`
	AccountID uint      `json:"account_id" gorm:"index;not null;comment:所属邮箱账号ID"`
	Name      string    `json:"name" gorm:"type:varchar(64);not null;comment:目录显示名称"`
	Slug      string    `json:"slug" gorm:"type:varchar(64);not null;uniqueIndex:idx_mail_folders_account_slug;comment:目录标识(写入mails.folder)"`
	SortOrder int       `json:"sort_order" gorm:"default:0;comment:排序(越小越靠前)"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Account *MailAccount `json:"account,omitempty" gorm:"foreignKey:AccountID"`
}

func (MailFolder) TableName() string { return "mail_folders" }

// MailRule 邮件归档规则（作用于指定邮箱账号收到的新邮件）
// 条件字段: from/subject/to/body/has_attachment; 操作: contains/not_contains/equals/starts_with/is_set
type MailRule struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"user_id" gorm:"index;not null;default:0;comment:所属用户ID(数据隔离)"`
	AccountID  uint      `json:"account_id" gorm:"index;not null;comment:作用的邮箱账号ID"`
	Name       string    `json:"name" gorm:"type:varchar(64);not null;default:'';comment:规则名称"`
	Field      string    `json:"field" gorm:"type:varchar(32);not null;comment:匹配字段(from/subject/to/body/has_attachment)"`
	Operator   string    `json:"operator" gorm:"type:varchar(32);not null;comment:操作(contains/not_contains/equals/starts_with/is_set)"`
	Value      string    `json:"value" gorm:"type:varchar(255);not null;default:'';comment:匹配值(操作为is_set时忽略)"`
	FolderSlug string    `json:"folder_slug" gorm:"type:varchar(64);not null;comment:命中后归入的目录slug"`
	Priority   int       `json:"priority" gorm:"default:0;comment:优先级(越大越先匹配)"`
	Enabled    bool      `json:"enabled" gorm:"not null;comment:是否启用"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (MailRule) TableName() string { return "mail_rules" }

// MailRuleRequest 规则创建/更新请求
type MailRuleRequest struct {
	Name       string `json:"name"`
	Field      string `json:"field"`
	Operator   string `json:"operator"`
	Value      string `json:"value"`
	FolderSlug string `json:"folder_slug"`
	Priority   int    `json:"priority"`
	Enabled    *bool  `json:"enabled"`
}

// MailFolderRequest 目录创建/更新请求
type MailFolderRequest struct {
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}
