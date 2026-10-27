// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"cosmomail/models"

	// 纯 Go SQLite 驱动（基于 modernc.org/sqlite，无需 CGO）
	"github.com/glebarez/sqlite"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Init 初始化 SQLite 数据库连接并执行自动迁移
func Init(dsn string) *gorm.DB {
	// 确保 data 目录存在
	dbDir := filepath.Dir(dsn)
	if dbDir != "." && dbDir != "" {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			log.Fatalf("❌ 无法创建数据目录 %s: %v", dbDir, err)
		}
	}

	// 按环境控制 SQL 日志：生产环境静默，开发环境输出 SQL
	logLevel := gormlogger.Silent
	if os.Getenv("COSMOMAIL_ENV") != "production" {
		logLevel = gormlogger.Info
	}

	// 连接 SQLite
	db, err := gorm.Open(sqlite.Open(dsn+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	})
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}

	// 获取底层数据库连接并配置
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("❌ 获取数据库实例失败: %v", err)
	}
	// 多邮箱并发同步时每个 Worker 都会写库（同步状态/邮件入库），
	// SQLite 单写者限制下并发写会互相 SQLITE_BUSY，表现为"多个邮箱只有一个能连通"。
	// 写连接串行化（1 个写连接排队）+ busy_timeout 兜底，保证所有账号的同步都能执行。
	// 注意 SQLite 单写者：只有 1 个连接能写，多连接同时写立刻 SQLITE_BUSY。
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	db.Exec("PRAGMA busy_timeout = 5000")

	// 自动迁移表结构
	if err := db.AutoMigrate(
		&models.MailAccount{},
		&models.Mail{},
		&models.Attachment{},
		&models.Webhook{},
		&models.WebhookLog{},
		&models.User{},
		&models.AppConfig{},
		&models.Draft{},
		&models.PushSubscription{},
		&models.MailboxSyncState{},
		&models.Contact{},
		&models.AppSetting{},
		&models.MailFolder{},
		&models.MailRule{},
	); err != nil {
		log.Fatalf("❌ 数据库迁移失败: %v", err)
	}

	// 多用户模式：users 表 username 唯一索引已由 AutoMigrate 建立，
	// 不再限制"只能存在一个管理员"（旧版单管理员唯一约束若存在则移除）。
	if err := db.Exec("DROP INDEX IF EXISTS idx_users_single_admin").Error; err != nil {
		log.Printf("⚠️  移除旧单管理员约束失败: %v", err)
	}

	// 旧版本升级兼容：邮箱账号的 email 唯一索引从单用户版的全局唯一改为 (user_id, email)
	// 组合唯一（AutoMigrate 只会新建索引不会删除旧索引），多用户模式下不同用户可各自添加同一邮箱。
	// 新组合索引由 AutoMigrate 建立为 idx_account_user_email；旧全局索引名按gorm默认规则匹配。
	// 注意必须先 AutoMigrate 建好新索引再删旧索引，否则唯一约束会出现空窗。
	for _, oldIdx := range []string{"idx_mail_accounts_email"} {
		if err := db.Exec("DROP INDEX IF EXISTS " + oldIdx).Error; err != nil {
			log.Printf("⚠️  移除旧邮箱唯一索引 %s 失败: %v", oldIdx, err)
		}
	}

	// 旧版本升级兼容：多用户版新增 is_admin 字段，历史用户默认全部 false。
	// 若没有任何管理员，则把最早注册的用户提升为管理员，否则无人能进入用户管理/注册开关。
	var totalUsers, adminUsers int64
	db.Model(&models.User{}).Count(&totalUsers)
	db.Model(&models.User{}).Where("is_admin = ?", true).Count(&adminUsers)
	if totalUsers > 0 && adminUsers == 0 {
		var firstUser models.User
		if err := db.Order("created_at ASC, id ASC").First(&firstUser).Error; err == nil {
			if err := db.Model(&firstUser).Update("is_admin", true).Error; err == nil {
				log.Printf("👑 已将最早注册的用户「%s」提升为管理员（旧版本升级兼容）", firstUser.Username)
			}
		}
	}

	// 迁移后清理：修复历史重复入库问题
	// 1. 删除 account_id+folder+message_uid 维度的重复记录（保留最早入库的一条）
	// 2. 将旧的带时间戳的 fallback message_id 更新为稳定格式
	// 3. 创建复合唯一索引 account_id+folder+message_uid
	// 4. 移除过宽的旧唯一索引（message_id 全局唯一）
	cleanupDuplicateMails(db)

	fmt.Println("✅ 数据库初始化成功:", dsn)
	return db
}

// EnsureSecuritySecrets 确保安全密钥存在：
//   - 首次启动时优先使用环境变量传入的密钥（COSMOMAIL_JWT_SECRET / COSMOMAIL_ENCRYPT_KEY），
//     未设置则自动生成随机密钥，最终持久化到数据库；
//   - 后续启动从数据库读取，但若环境变量有新值则以环境变量为准（覆盖数据库值）。
func EnsureSecuritySecrets(db *gorm.DB, jwtSecret, encryptionKey *string) {
	// 优先从环境变量读取用户指定的密钥
	envJWT := os.Getenv("COSMOMAIL_JWT_SECRET")
	envEncKey := os.Getenv("COSMOMAIL_ENCRYPT_KEY")

	var cfg models.AppConfig
	result := db.First(&cfg)

	if result.Error != nil {
		// 首次启动：环境变量 > 自动生成
		jwtSec := envJWT
		encKey := envEncKey
		var err error

		if jwtSec == "" {
			jwtSec, err = models.GenerateRandomKey(32)
			if err != nil {
				log.Fatalf("❌ 生成 JWT 密钥失败: %v", err)
			}
			log.Println("🔑 JWT 密钥：已自动生成随机密钥")
		} else {
			log.Println("🔑 JWT 密钥：从环境变量 COSMOMAIL_JWT_SECRET 读取")
		}

		if encKey == "" {
			encKey, err = models.GenerateRandomKey(32)
			if err != nil {
				log.Fatalf("❌ 生成加密密钥失败: %v", err)
			}
			log.Println("🔐 加密密钥：已自动生成随机密钥")
		} else {
			log.Println("🔐 加密密钥：从环境变量 COSMOMAIL_ENCRYPT_KEY 读取")
		}

		cfg = models.AppConfig{
			JWTSecret:     jwtSec,
			EncryptionKey: encKey,
		}
		if err := db.Create(&cfg).Error; err != nil {
			log.Fatalf("❌ 保存安全配置失败: %v", err)
		}

		*jwtSecret = jwtSec
		*encryptionKey = encKey
	} else {
		// 已有记录：环境变量 > 数据库存储
		useJWT := cfg.JWTSecret
		useEncKey := cfg.EncryptionKey
		sourceJWT := "数据库"
		sourceEncKey := "数据库"

		if envJWT != "" && envJWT != cfg.JWTSecret {
			useJWT = envJWT
			sourceJWT = "环境变量"
			// 同步更新数据库，保证下次启动一致
			cfg.JWTSecret = envJWT
			db.Save(&cfg)
		}

		if envEncKey != "" && envEncKey != cfg.EncryptionKey {
			useEncKey = envEncKey
			sourceEncKey = "环境变量"
			cfg.EncryptionKey = envEncKey
			db.Save(&cfg)
		}

		*jwtSecret = useJWT
		*encryptionKey = useEncKey
		log.Printf("🔐 安全密钥已加载（JWT 来源：%s，加密密钥 来源：%s）", sourceJWT, sourceEncKey)
	}
}

// cleanupDuplicateMails 迁移后清理函数：修复历史重复入库问题
//
// 问题背景：旧版 fallback Message-ID 使用了 time.Now()，导致同一封邮件
// 每次同步生成不同 message_id，去重失效，同一封邮件被反复入库。
//
// 本函数执行以下步骤：
//  1. 删除 account_id + folder + message_uid 维度的重复记录（保留最早入库的一条）
//  2. 将旧的带时间戳的 fallback message_id 更新为新的稳定格式
//  3. 创建复合唯一索引 idx_mails_account_folder_uid
//  4. 移除过宽的旧唯一索引（message_id 全局唯一），避免不同账号收到相同 Message-ID 时冲突
func cleanupDuplicateMails(db *gorm.DB) {
	// 检查 mails 表是否存在
	var tableExists int64
	db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='mails'").Scan(&tableExists)
	if tableExists == 0 {
		return // 新数据库，无需清理
	}

	// 步骤1：删除 account_id + folder + message_uid 维度的重复记录（保留最小 id）
	result := db.Exec(`
		DELETE FROM mails
		WHERE id NOT IN (
			SELECT MIN(id) FROM mails
			WHERE message_uid > 0
			GROUP BY account_id, folder, message_uid
		)
		AND message_uid > 0
	`)
	if result.Error != nil {
		log.Printf("⚠️  [迁移] 清理重复邮件记录失败: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("🧹 [迁移] 已清理 %d 条重复邮件记录", result.RowsAffected)
	}

	// 步骤2：将旧的 fallback message_id 更新为新的稳定格式
	// 旧格式: <auto-{uid}-{timestamp}@proxy>  →  新格式: <auto-account-{aid}-folder-{folder}-uid-{uid}@cosmomail>
	result = db.Exec(`
		UPDATE mails
		SET message_id = '<auto-account-' || account_id || '-folder-' || COALESCE(folder, 'inbox') || '-uid-' || message_uid || '@cosmomail>'
		WHERE message_id LIKE '<auto-%@proxy>'
	`)
	if result.Error != nil {
		log.Printf("⚠️  [迁移] 更新旧 IMAP fallback message_id 失败: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("🔄 [迁移] 已更新 %d 条旧格式 IMAP fallback message_id", result.RowsAffected)
	}

	// 旧格式: <pop3-{seq}-{timestamp}@proxy>  →  新格式: <pop3-account-{aid}-seq-{seq}@cosmomail>
	result = db.Exec(`
		UPDATE mails
		SET message_id = '<pop3-account-' || account_id || '-seq-' || message_uid || '@cosmomail>'
		WHERE message_id LIKE '<pop3-%@proxy>'
	`)
	if result.Error != nil {
		log.Printf("⚠️  [迁移] 更新旧 POP3 fallback message_id 失败: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("🔄 [迁移] 已更新 %d 条旧格式 POP3 fallback message_id", result.RowsAffected)
	}

	// 步骤3：移除过宽的旧唯一索引（message_id 全局唯一）
	// GORM 默认命名为 uniq_mails_message_id
	db.Exec("DROP INDEX IF EXISTS uniq_mails_message_id")

	// 步骤4：创建复合唯一索引（如果不存在）
	// 确保 account_id + folder + message_uid 三元组唯一，防止重复入库
	result = db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_mails_account_folder_uid
		ON mails(account_id, folder, message_uid)
	`)
	if result.Error != nil {
		log.Printf("⚠️  [迁移] 创建复合唯一索引失败（可能存在残留重复数据，不影响运行）: %v", result.Error)
		log.Printf("⚠️  [迁移] 建议手动执行: DELETE FROM mails WHERE id NOT IN (SELECT MIN(id) FROM mails WHERE message_uid > 0 GROUP BY account_id, folder, message_uid) AND message_uid > 0; 然后重启")
	} else {
		log.Printf("✅ [迁移] 复合唯一索引 idx_mails_account_folder_uid 已就绪")
	}
}
