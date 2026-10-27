// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package services

import (
	"path/filepath"
	"testing"

	"novamail/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// 回归测试：一键已读/批量已读等 Update 语句不得依赖 JOIN（SQLite 的 UPDATE 不支持 JOIN，
// 且 GORM 生成 UPDATE 时会丢弃 Joins，导致 "no such column: mail_accounts.user_id"）
func TestMarkAllAsReadRepro(t *testing.T) {
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "t.db")), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	if err := db.AutoMigrate(&models.Mail{}, &models.MailAccount{}); err != nil { t.Fatal(err) }
	acct := models.MailAccount{Email: "a@x.com", UserID: 1}
	db.Create(&acct)
	db.Create(&models.Mail{AccountID: acct.ID, Subject: "s", IsRead: false})
	s := NewMailService(db, nil)
	n, err := s.MarkAllAsRead(1, models.MailListFilter{})
	t.Logf("updated=%d err=%v", n, err)
	if err != nil || n != 1 {
		t.Fatalf("一键已读回归: updated=%d err=%v", n, err)
	}

	// 批量已读 / 单封已读 / 星标 同样不得报错
	if r := s.BatchMarkAsRead(1, []uint{1}, false); !r.Success {
		t.Fatalf("批量已读回归失败: %+v", r)
	}
	if err := s.MarkAsRead(1, 1, true); err != nil {
		t.Fatalf("单封已读回归失败: %v", err)
	}
	if err := s.MarkAsStarred(1, 1, true); err != nil {
		t.Fatalf("星标回归失败: %v", err)
	}
}
