// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package contacts

import (
	"path/filepath"
	"testing"

	"cosmomail/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "t.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Contact{}, &models.MailAccount{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestParseVCard(t *testing.T) {
	vc := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Alice Zhang\r\nN:Zhang;Alice;;;\r\nEMAIL;TYPE=WORK:alice@example.com\r\nEMAIL;TYPE=HOME:ALICE2@Example.COM\r\nEND:VCARD"
	name, emails := parseVCard(vc)
	if name != "Alice Zhang" {
		t.Fatalf("name=%q", name)
	}
	if len(emails) != 2 || emails[0] != "alice@example.com" || emails[1] != "ALICE2@Example.COM" {
		t.Fatalf("emails=%v", emails)
	}
}

func TestParseVCardGroupedEmail(t *testing.T) {
	vc := "BEGIN:VCARD\r\nFN:Bob\r\nitem1.EMAIL;TYPE=INTERNET:bob@example.com\r\nEND:VCARD"
	_, emails := parseVCard(vc)
	if len(emails) != 1 || emails[0] != "bob@example.com" {
		t.Fatalf("emails=%v", emails)
	}
}

func TestUpsertContactDedup(t *testing.T) {
	db := newTestDB(t)
	s := NewSyncService(db)
	if err := s.UpsertContact(1, 2, "a@x.com", "A", "carddav"); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertContact(1, 2, models.NormalizeEmail("A@X.com "), "A2", "carddav"); err != nil {
		t.Fatal(err)
	}
	var cnt int64
	db.Model(&models.Contact{}).Count(&cnt)
	if cnt != 1 {
		t.Fatalf("去重失败 count=%d", cnt)
	}
	var ct models.Contact
	db.First(&ct)
	if ct.UseCount != 2 || ct.Source != "carddav" || ct.Name != "A" {
		t.Fatalf("upsert 字段异常: %+v", ct)
	}
}

func TestSyncCardDAVDisabled(t *testing.T) {
	db := newTestDB(t)
	s := NewSyncService(db)
	if _, err := s.SyncCardDAV(1, 1, CardDAVConfig{Enabled: false}); err == nil {
		t.Fatal("未启用时应报错")
	}
}
