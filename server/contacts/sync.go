// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package contacts

import (
	"log"
	"net/mail"
	"strings"
	"time"

	"novamail/models"

	"gorm.io/gorm"
)

// SyncService 邮件联系人提取 —— 从已同步的邮件中提取发件人/收件人/抄送，
// 按邮箱账号归档形成该账号的通讯录（方案A：所有邮箱服务商通用，无需额外配置）
type SyncService struct {
	db *gorm.DB
}

// NewSyncService 创建联系人提取服务
func NewSyncService(db *gorm.DB) *SyncService {
	return &SyncService{db: db}
}

// ExtractFromMail 从一封已入库的邮件提取联系人并 upsert。
// senderAccountID 为该邮件所属邮箱账号 ID；ownerID 为数据隔离的用户 ID。
// 提取范围：From（对方发来的）、To、Cc（账号发出的或抄送到的）。
// 自身账号邮箱地址会被跳过，避免把自己记成联系人。
func (s *SyncService) ExtractFromMail(ownerID, senderAccountID uint, accountEmail, from, to, cc string, sentAt time.Time) {
	now := time.Now()
	self := models.NormalizeEmail(accountEmail)

	s.upsert(ownerID, senderAccountID, from, self, sentAt, now)
	for _, list := range []string{to, cc} {
		s.upsertList(ownerID, senderAccountID, list, self, sentAt, now)
	}
}

func (s *SyncService) upsertList(ownerID, accountID uint, list, self string, seenAt, now time.Time) {
	for _, item := range s.splitAddresses(list) {
		s.upsert(ownerID, accountID, item, self, seenAt, now)
	}
}

// splitAddresses 拆分 "Name <a@x>, b@y" 形式的地址列表
func (s *SyncService) splitAddresses(list string) []string {
	if strings.TrimSpace(list) == "" {
		return nil
	}
	return strings.Split(list, ",")
}

func (s *SyncService) upsert(ownerID, accountID uint, raw, self string, seenAt, now time.Time) {
	item := strings.TrimSpace(raw)
	if item == "" {
		return
	}
	email, name := parseOneAddress(item)
	email = models.NormalizeEmail(email)
	if email == "" || !strings.Contains(email, "@") || email == self {
		return
	}

	var existing models.Contact
	err := s.db.Where("user_id = ? AND email = ?", ownerID, email).First(&existing).Error
	if err != nil {
		if err := s.db.Create(&models.Contact{
			UserID: ownerID, Email: email, Name: name,
			AccountID: accountID, LastUsedAt: seenAt, UseCount: 1,
			Source: "sync",
		}).Error; err != nil {
			// 并发同步可能撞唯一索引，转为更新
			s.touch(existing, email, ownerID, name, seenAt, now)
		}
		return
	}
	s.touch(existing, email, ownerID, name, seenAt, now)
}

// touch 更新已有联系人：累加次数、刷新时间、姓名为空时补齐、AccountID 首次绑定后不覆盖
func (s *SyncService) touch(existing models.Contact, email string, ownerID uint, name string, seenAt, now time.Time) {
	updates := map[string]interface{}{
		"use_count":    existing.UseCount + 1,
		"last_used_at": seenAt,
	}
	if existing.Name == "" && name != "" {
		updates["name"] = name
	}
	if err := s.db.Model(&models.Contact{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
		log.Printf("[WARN] 更新联系人 %s 失败: %v", email, err)
	}
}

// parseOneAddress 解析单个 "Name <a@x>" 或 "a@x"，返回 (email, name)
func parseOneAddress(item string) (string, string) {
	if addr, err := mail.ParseAddress(item); err == nil {
		return addr.Address, addr.Name
	}
	// 兜底：尖括号提取
	if i := strings.Index(item, "<"); i >= 0 && strings.HasSuffix(item, ">") {
		return item[i+1 : len(item)-1], strings.TrimSpace(item[:i])
	}
	return item, ""
}

// SyncAccountContacts 为指定账号做一次全量联系人提取（从本地已同步邮件中）。
// 返回新提取的联系人数量（近似：受影响行数）。
func (s *SyncService) SyncAccountContacts(ownerID, accountID uint, accountEmail string) (int64, error) {
	var mails []models.Mail
	if err := s.db.Select("id", "from", "\"to\"", "cc", "sent_at").
		Where("account_id = ?", accountID).
		Order("sent_at ASC").Find(&mails).Error; err != nil {
		return 0, err
	}
	var affected int64
	for _, m := range mails {
		before := s.countFor(ownerID)
		s.ExtractFromMail(ownerID, accountID, accountEmail, m.From, m.To, m.Cc, m.SentAt)
		affected += s.countFor(ownerID) - before
	}
	return affected, nil
}

func (s *SyncService) countFor(ownerID uint) int64 {
	var n int64
	s.db.Model(&models.Contact{}).Where("user_id = ?", ownerID).Count(&n)
	return n
}

// ListByAccount 按账号查询通讯录；accountID 为 0 时返回该用户全部账号的联系人。
func (s *SyncService) ListByAccount(ownerID, accountID uint, keyword string, limit int) ([]models.Contact, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	query := s.db.Where("user_id = ?", ownerID)
	if accountID > 0 {
		query = query.Where("account_id = ?", accountID)
	}
	if keyword != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(keyword)) + "%"
		query = query.Where("(email LIKE ? OR name LIKE ?)", kw, kw)
	}
	var contacts []models.Contact
	err := query.Order("last_used_at DESC").Limit(limit).Find(&contacts).Error
	return contacts, err
}
