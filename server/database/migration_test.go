package database

import (
	"path/filepath"
	"testing"

	"novamail/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestOldDBMigration(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "old.db")), &gorm.Config{})
	// 模拟旧库：全局唯一索引
	db.Exec("CREATE TABLE mail_accounts (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL DEFAULT 0, email VARCHAR(255) NOT NULL, name VARCHAR(100) NOT NULL DEFAULT '', protocol VARCHAR(10) NOT NULL DEFAULT 'imap', username VARCHAR(255) NOT NULL DEFAULT '', password TEXT NOT NULL DEFAULT '', auth_type VARCHAR(20) DEFAULT '', status VARCHAR(20) NOT NULL DEFAULT 'active', imap_host VARCHAR(255) NOT NULL DEFAULT '', port INTEGER DEFAULT 993, created_at DATETIME, updated_at DATETIME)")
	db.Exec("CREATE UNIQUE INDEX idx_mail_accounts_email ON mail_accounts(email)")
	db.Exec("INSERT INTO mail_accounts (user_id, email) VALUES (1, 'a@x.com')")

	// AutoMigrate 加新列/新索引（glebarez 对已有表会补索引）
	if err := db.AutoMigrate(&models.MailAccount{}); err != nil {
		t.Fatal(err)
	}
	// 生产迁移逻辑：删旧索引
	if err := db.Exec("DROP INDEX IF EXISTS idx_mail_accounts_email").Error; err != nil {
		t.Fatal(err)
	}

	// 另一个用户现在可以添加相同邮箱
	if err := db.Exec("INSERT INTO mail_accounts (user_id, email, name, username, password, imap_host) VALUES (2, 'a@x.com', 'n2', 'u2', 'pw', 'imap.x.com')").Error; err != nil {
		t.Fatalf("迁移后不同用户添加相同邮箱失败: %v", err)
	}
	t.Log("✅ 旧库迁移后多用户可各自添加同一邮箱")
}
