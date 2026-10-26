// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package models

import "time"

// User 用户模型（多用户模式）
type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"uniqueIndex;size:64;not null"`
	PasswordHash string    `json:"-" gorm:"size:255;not null;column:password_hash"` // bcrypt 哈希，不序列化输出
	IsAdmin      bool      `json:"is_admin" gorm:"default:false;not null;column:is_admin;comment:是否管理员"` // 首个注册用户自动成为管理员
	Disabled     bool      `json:"disabled" gorm:"default:false;not null;column:disabled;comment:是否被禁用"` // 禁用后无法登录，API Token 同样失效
	ApiToken     string    `json:"-" gorm:"size:64;index;column:api_token;comment:外部 API 访问令牌(每用户唯一)"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

// ---- 请求 DTO ----

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// RegisterRequest 注册请求（仅首次初始化时可用）
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=32"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// ---- 响应 DTO ----

// LoginResponse 登录响应（含 Token）
type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
}

// AuthStatusResponse 认证状态响应（用于判断是否需要注册/是否允许注册）
type AuthStatusResponse struct {
	SetupRequired       bool   `json:"setup_required"`        // true = 需要注册（尚无任何用户）
	RegistrationEnabled bool   `json:"registration_enabled"`  // 管理员是否开放了注册
	Message             string `json:"message"`
}
