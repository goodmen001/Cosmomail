// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

// Package contacts —— CardDAV (RFC 6352) 轻量客户端：从用户自建/服务商通讯录同步联系人。
// 不引入额外依赖，直接使用标准库 HTTP + XML 实现 REPORT addressbook-query。
package contacts

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"novamail/models"
)

// CardDAVConfig 用户的 CardDAV 连接配置（存于邮箱账号上）
type CardDAVConfig struct {
	Enabled  bool
	URL      string // 地址簿集合 URL，如 https://server/.well-known/carddav/user/
	Username string
	Password string
}

// carddavMultistatus DAV:multistatus 响应
type carddavMultistatus struct {
	Responses []carddavResponse `xml:"response"`
}

type carddavResponse struct {
	Href string `xml:"href"`
	Propstat []struct {
		Prop struct {
			AddressData string `xml:"address-data"`
		} `xml:"prop"`
	} `xml:"propstat"`
}

const carddavReportBody = `<?xml version="1.0" encoding="utf-8"?>
<C:addressbook-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav">
  <D:prop>
    <D:getetag/>
    <C:address-data>
      <C:prop><C:fn/><C:n/><C:email/></C:prop>
    </C:address-data>
  </D:prop>
</C:addressbook-query>`

// SyncCardDAV 拉取 CardDAV 地址簿中所有 vCard，提取邮箱联系人并入库
func (s *SyncService) SyncCardDAV(ownerID uint, accountID uint, cfg CardDAVConfig) (int, error) {
	if !cfg.Enabled || cfg.URL == "" {
		return 0, fmt.Errorf("CardDAV 未启用或未配置")
	}
	vcards, err := fetchVCards(cfg)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, vc := range vcards {
		name, emails := parseVCard(vc)
		for _, email := range emails {
			email = models.NormalizeEmail(email)
			if email == "" || !strings.Contains(email, "@") {
				continue
			}
			if err := s.UpsertContact(ownerID, accountID, email, name, "carddav"); err != nil {
				continue
			}
			n++
		}
	}
	return n, nil
}

// fetchVCards 执行 REPORT addressbook-query，返回全部 vCard 文本
func fetchVCards(cfg CardDAVConfig) ([]string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("重定向过多")
			}
			return nil
		},
	}
	req, err := http.NewRequest("REPORT", cfg.URL, strings.NewReader(carddavReportBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	req.Header.Set("Depth", "1")
	req.SetBasicAuth(cfg.Username, cfg.Password)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("地址簿 URL 不存在(404)，请检查 CardDAV 地址")
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("CardDAV 认证失败(401)，请检查用户名密码")
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("CardDAV 服务器返回 %d: %s", resp.StatusCode, string(body))
	}

	var ms carddavMultistatus
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return nil, fmt.Errorf("解析 CardDAV 响应失败: %w", err)
	}
	var out []string
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			if ps.Prop.AddressData != "" {
				out = append(out, ps.Prop.AddressData)
			}
		}
	}
	return out, nil
}

// parseVCard 极简 vCard 解析：取 FN（显示名）与所有 EMAIL 行
func parseVCard(vcard string) (string, []string) {
	var name string
	var emails []string
	for _, line := range strings.Split(vcard, "\n") {
		line = strings.TrimSpace(line)
		// 去掉 vCard 分组前缀，如 "item1.EMAIL:..." → "EMAIL:..."
		if dot := strings.Index(line, "."); dot > 0 && dot < strings.Index(line+":", ":") {
			if strings.HasPrefix(strings.ToUpper(line[dot+1:]), "EMAIL") || strings.HasPrefix(strings.ToUpper(line[dot+1:]), "FN") {
				line = line[dot+1:]
			}
		}
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "FN:") && name == "":
			name = strings.TrimSpace(line[3:])
		case strings.HasPrefix(upper, "EMAIL"):
			// EMAIL;TYPE=...:addr@example.com
			if idx := strings.Index(line, ":"); idx >= 0 {
				raw := strings.TrimSpace(line[idx+1:])
				// 有些 vCard 组有分组前缀 item1.EMAIL:
				if d, _, err := mime.ParseMediaType(raw); err == nil && d != "" {
					raw = d
				}
				if raw != "" {
					emails = append(emails, raw)
				}
			}
		}
	}
	return name, emails
}

// ---- SyncService 补充方法 ----

// UpsertContact 新增或触碰联系人（供 CardDAV / 其他来源使用）
func (s *SyncService) UpsertContact(ownerID, accountID uint, email, name, source string) error {
	var ct models.Contact
	err := s.db.Where("user_id = ? AND email = ?", ownerID, email).First(&ct).Error
	if err != nil {
		now := time.Now()
		return s.db.Create(&models.Contact{
			UserID: ownerID, Email: email, Name: name,
			AccountID: accountID, LastUsedAt: now, UseCount: 1, Source: source,
		}).Error
	}
	updates := map[string]interface{}{
		"last_used_at": time.Now(),
		"use_count":    ct.UseCount + 1,
	}
	if name != "" && ct.Name == "" {
		updates["name"] = name
	}
	return s.db.Model(&ct).Updates(updates).Error
}

var _ = bytes.Equal
