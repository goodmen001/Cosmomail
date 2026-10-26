// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"cosmomail/models"

	"gorm.io/gorm"
)

// GotifyService Gotify 新邮件通知服务（每用户独立配置，存 app_settings）
type GotifyService struct {
	db *gorm.DB

	mu      sync.RWMutex
	cached  map[uint]models.GotifyConfig // userID -> 配置缓存
	client  *http.Client
}

// gotifySettingKey 每用户配置键
func gotifySettingKey(userID uint) string {
	return fmt.Sprintf("gotify_config_%d", userID)
}

// NewGotifyService 创建 Gotify 服务
func NewGotifyService(db *gorm.DB) *GotifyService {
	return &GotifyService{
		db:     db,
		cached: make(map[uint]models.GotifyConfig),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetConfig 读取用户 Gotify 配置（URL + Token + 是否启用）
func (s *GotifyService) GetConfig(userID uint) models.GotifyConfig {
	s.mu.RLock()
	if cfg, ok := s.cached[userID]; ok {
		s.mu.RUnlock()
		return cfg
	}
	s.mu.RUnlock()

	var setting models.AppSetting
	cfg := models.GotifyConfig{}
	if err := s.db.Where("`key` = ?", gotifySettingKey(userID)).First(&setting).Error; err == nil {
		_ = json.Unmarshal([]byte(setting.Value), &cfg)
	}
	s.mu.Lock()
	s.cached[userID] = cfg
	s.mu.Unlock()
	return cfg
}

// SaveConfig 保存用户 Gotify 配置
func (s *GotifyService) SaveConfig(userID uint, cfg models.GotifyConfig) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	setting := models.AppSetting{Key: gotifySettingKey(userID), Value: string(raw), UpdatedAt: time.Now()}
	if err := s.db.Save(&setting).Error; err != nil {
		return err
	}
	s.mu.Lock()
	s.cached[userID] = cfg
	s.mu.Unlock()
	return nil
}

// Test 发送测试消息，返回错误信息（nil 表示成功）
func (s *GotifyService) Test(cfg models.GotifyConfig) error {
	return s.send(cfg, "Cosmo Mail 测试通知", "如果你看到这条消息，说明 Gotify 配置成功 🎉", 5)
}

// NotifyMailReceived 新邮件到达时推送（供 Worker 调用；配置无效或未启用时静默跳过）
func (s *GotifyService) NotifyMailReceived(userID uint, title, body string) {
	cfg := s.GetConfig(userID)
	if !cfg.Enabled || cfg.URL == "" || cfg.Token == "" {
		return
	}
	go func() {
		if err := s.send(cfg, title, body, 5); err != nil {
			log.Printf("[Gotify] 推送失败 (userID=%d): %v", userID, err)
		} else {
			log.Printf("[Gotify] ✓ 推送成功 (userID=%d)", userID)
		}
	}()
}

// send 调用 Gotify /message 接口
func (s *GotifyService) send(cfg models.GotifyConfig, title, body string, priority int) error {
	url := strings.TrimSuffix(strings.TrimSpace(cfg.URL), "/")
	if url == "" || cfg.Token == "" {
		return fmt.Errorf("Gotify URL 或 Token 未配置")
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"title":    title,
		"message":  body,
		"priority": priority,
	})
	req, err := http.NewRequest("POST", url+"/message?token="+cfg.Token, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}
