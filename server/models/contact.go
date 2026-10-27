// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package models

import (
	"strings"
	"time"
)

// Contact 联系人（收件人）记录 —— 记录发信收件人与邮件同步提取的联系人并去重
type Contact struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index;not null;comment:所属用户ID"`
	Email     string    `json:"email" gorm:"type:varchar(255);not null;uniqueIndex:idx_contact_user_email;comment:联系人邮箱"`
	Name      string    `json:"name" gorm:"type:varchar(100);comment:联系人名称(如有)"`
	AccountID uint      `json:"account_id" gorm:"index;default:0;comment:关联的邮箱账号ID(发信账号或提取来源账号)"`
	LastUsedAt time.Time `json:"last_used_at" gorm:"comment:最近一次出现时间"`
	UseCount  int64     `json:"use_count" gorm:"default:1;comment:出现次数"`
	Source    string    `json:"source" gorm:"type:varchar(20);default:'send';comment:来源(send=发信记录/sync=邮件同步提取/carddav=通讯录同步)"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Contact) TableName() string {
	return "contacts"
}

// NormalizeEmail 归一化邮箱地址（小写、去空白），用于去重
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// AppSetting 系统设置（键值对，单行存储一个设置项）
type AppSetting struct {
	Key       string    `json:"key" gorm:"primaryKey;type:varchar(64)"`
	Value     string    `json:"value" gorm:"type:text;not null"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (AppSetting) TableName() string {
	return "app_settings"
}

// 设置键名常量
const (
	SettingKeyRegistrationEnabled = "registration_enabled" // 是否开放注册："true"/"false"
)

// ---- 用户相关 DTO 扩展 ----

// RegisterEnabledRequest 管理员切换注册开关
type RegisterEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// UserListItem 用户管理列表项
type UserListItem struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	IsAdmin   bool      `json:"is_admin"`
	Disabled  bool      `json:"disabled"`
	HasToken  bool      `json:"has_token"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateUserRequest 管理员手动创建用户
type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

// SetUserDisabledRequest 管理员禁用/启用用户
type SetUserDisabledRequest struct {
	Disabled bool `json:"disabled"`
}

// GotifyConfig 用户的 Gotify 通知配置
type GotifyConfig struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	Token   string `json:"token"`
}

// ChangePasswordRequest 修改密码
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
}
