// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

// Package oauth2token 提供 OAuth2 Access Token 的统一获取与自动刷新能力，
// 供 IMAP 收信与 SMTP 发信共用。刷新后的 Token 会持久化回数据库，
// 避免每次发信/同步都重新刷新。
package oauth2token

import (
	"context"
	"log"
	"time"

	"cosmomail/models"
	"cosmomail/oauth2"

	"gorm.io/gorm"
)

// ResolveAccessToken 获取账号当前有效的 Access Token。
// - Token 未过期（提前 2 分钟）且已缓存时直接复用；
// - 过期则使用 RefreshToken 刷新，并将新 Token 写回数据库。
// 非 OAuth2 账号返回空字符串。
func ResolveAccessToken(db *gorm.DB, account *models.MailAccount) string {
	providerName := oauth2.ParseAuthType(account.AuthType)
	if providerName == "" {
		return ""
	}

	provider, clientID, err := oauth2.ResolveProviderAndClientID(providerName, account.CustomClientId)
	if err != nil {
		log.Printf("[OAuth2] Provider 解析失败 (%s): %v", account.Email, err)
		return ""
	}

	// 提前 2 分钟视为过期；Password 字段在 OAuth2 模式下复用缓存 Access Token
	if account.TokenExpiresAt != nil {
		expiresAt := account.TokenExpiresAt.Add(-2 * time.Minute)
		if time.Now().Before(expiresAt) && account.Password != "" {
			return account.Password
		}
	}

	if account.RefreshToken == "" {
		log.Printf("[WARN] OAuth2 账号 %s 无 RefreshToken，需要用户重新授权", account.Email)
		return ""
	}

	log.Printf("🔄 正在刷新 AccessToken (%s@%s)...", account.Email, provider.Name())
	tokenResp, err := provider.RefreshAccessToken(context.Background(), clientID, account.RefreshToken)
	if err != nil {
		log.Printf("❌ RefreshToken 刷新失败 (%s): %v", account.Email, err)
		return ""
	}

	// 持久化刷新结果（Access Token 存入 Password 字段，RefreshToken 可能被轮换）
	updates := map[string]interface{}{
		"password":         tokenResp.AccessToken,
		"token_expires_at": time.Now().Add(tokenResp.ExpiresIn),
	}
	if tokenResp.RefreshToken != "" {
		updates["refresh_token"] = tokenResp.RefreshToken
	}
	if err := db.Model(&models.MailAccount{}).Where("id = ?", account.ID).Updates(updates).Error; err != nil {
		log.Printf("[WARN] AccessToken 持久化失败 (%s): %v", account.Email, err)
	}

	// 同步运行时缓存
	account.Password = tokenResp.AccessToken
	expiresAt := time.Now().Add(tokenResp.ExpiresIn)
	account.TokenExpiresAt = &expiresAt
	if tokenResp.RefreshToken != "" {
		account.RefreshToken = tokenResp.RefreshToken
	}

	log.Printf("✅ AccessToken 刷新成功: %s (有效期 %v)", account.Email, tokenResp.ExpiresIn)
	return tokenResp.AccessToken
}
