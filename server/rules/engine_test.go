// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package rules

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
	if err := db.AutoMigrate(&models.MailRule{}, &models.Mail{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func mail(from, subject string) *models.Mail {
	return &models.Mail{From: from, Subject: subject}
}

func TestApplyPriorityAndMatch(t *testing.T) {
	db := newTestDB(t)
	rules := []models.MailRule{
		{AccountID: 1, Field: "subject", Operator: "contains", Value: "invoice", FolderSlug: "billing", Priority: 1, Enabled: true},
		{AccountID: 1, Field: "from", Operator: "contains", Value: "boss@corp.com", FolderSlug: "vip", Priority: 10, Enabled: true},
		{AccountID: 1, Field: "subject", Operator: "contains", Value: "订阅", FolderSlug: "news", Priority: 5, Enabled: false}, // 禁用不参与
	}
	for i := range rules {
		if err := db.Create(&rules[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	e := NewEngine(db)

	// 高优先级命中
	if r := e.Apply(1, mail("boss@corp.com", "hello")); r.FolderSlug != "vip" {
		t.Fatalf("expected vip, got %+v", r)
	}
	// 低优先级规则兜底
	if r := e.Apply(1, mail("a@b.com", "your invoice #123")); r.FolderSlug != "billing" {
		t.Fatalf("expected billing, got %+v", r)
	}
	// 禁用规则不生效
	if r := e.Apply(1, mail("a@b.com", "每日订阅推送")); r.FolderSlug != "" {
		t.Fatalf("disabled rule should not match, got %+v", r)
	}
	// 无命中 → inbox
	if r := e.Apply(1, mail("a@b.com", "hi")); r.FolderSlug != "" {
		t.Fatalf("expected no match, got %+v", r)
	}
	// 账号隔离
	if r := e.Apply(2, mail("boss@corp.com", "hello")); r.FolderSlug != "" {
		t.Fatalf("account isolation failed: %+v", r)
	}
}

func TestOperators(t *testing.T) {
	db := newTestDB(t)
	cases := []struct {
		rule  models.MailRule
		mail  *models.Mail
		match bool
	}{
		{models.MailRule{Field: "from", Operator: "equals", Value: "a@b.com"}, mail("A@B.COM", "x"), true},
		{models.MailRule{Field: "from", Operator: "equals", Value: "a@b.com"}, mail("b@b.com", "x"), false},
		{models.MailRule{Field: "from", Operator: "not_contains", Value: "spam"}, mail("good@x.com", "y"), true},
		{models.MailRule{Field: "from", Operator: "starts_with", Value: "noreply"}, mail("noreply@x.com", "y"), true},
		{models.MailRule{Field: "has_attachment", Operator: "is_set"}, &models.Mail{HasAttachment: true}, true},
		{models.MailRule{Field: "has_attachment", Operator: "is_set"}, &models.Mail{HasAttachment: false}, false},
		{models.MailRule{Field: "to", Operator: "contains", Value: "team"}, mail("a@b.com", "y"), false}, // to 为空
	}
	for i, c := range cases {
		c.rule.AccountID, c.rule.Enabled = 1, true
		if err := db.Create(&c.rule).Error; err != nil {
			t.Fatal(err)
		}
		got := NewEngine(db).Apply(1, c.mail)
		if (got.RuleID != 0) != c.match {
			t.Fatalf("case %d: rule=%+v mail=%+v got=%+v", i, c.rule, c.mail, got)
		}
		db.Delete(&c.rule)
	}
}
