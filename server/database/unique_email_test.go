// SPDX-License-Identifier: AGPL-3.0-or-later
package database

import (
	"path/filepath"
	"testing"

	"novamail/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAccountEmailUniquePerUser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "t.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.MailAccount{}); err != nil {
		t.Fatal(err)
	}
	db.Exec("DROP INDEX IF EXISTS idx_mail_accounts_email")

	a1 := models.MailAccount{UserID: 1, Email: "same@x.com", Name: "u1"}
	a2 := models.MailAccount{UserID: 2, Email: "same@x.com", Name: "u2"}
	if err := db.Create(&a1).Error; err != nil {
		t.Fatalf("用户1创建失败: %v", err)
	}
	if err := db.Create(&a2).Error; err != nil {
		t.Fatalf("不同用户添加相同邮箱应允许，实际报错: %v", err)
	}
	var dup models.MailAccount
	dup.UserID, dup.Email = 1, "same@x.com"
	if err := db.Create(&dup).Error; err == nil {
		t.Fatal("同一用户重复添加相同邮箱应报唯一约束错误")
	}
	t.Log("✅ 邮箱唯一性已按 (user_id, email) 组合生效")
}
