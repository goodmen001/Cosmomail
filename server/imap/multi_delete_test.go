// 临时验证测试（不提交）：首屏全量同步（seq 范围抓取）+ DeleteAfterDownload
// 验证多封邮件场景下，删除远程邮件是否会导致本轮抓取漏信/错删。
package imap

import (
	"fmt"
	"net"
	"path/filepath"
	"testing"

	"novamail/config"
	"novamail/models"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestMultiMailDeleteAfterDownload(t *testing.T) {
	mem := imapmemserver.New()
	u := imapmemserver.NewUser("multi@test", "pass")
	u.Create("INBOX", nil)
	// 先放 5 封邮件再 AddUser，确保一次全量同步抓到多封
	for i := 1; i <= 5; i++ {
		raw := fmt.Sprintf("From: S%d <s%d@x.com>\r\nTo: b@x.com\r\nSubject: M%d\r\nDate: Mon, 26 Oct 2026 10:0%d:00 +0800\r\nMessage-ID: <m%d@x.com>\r\nContent-Type: text/plain\r\n\r\nbody%d\r\n", i, i, i, i%10, i, i)
		if _, err := u.Append("INBOX", newLiteral([]byte(raw)), &imap.AppendOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	mem.AddUser(u)
	srv := imapserver.New(&imapserver.Options{
		NewSession:   func(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) { return mem.NewSession(), nil, nil },
		InsecureAuth: true,
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(ln)
	defer srv.Close()
	addr := ln.Addr().String()

	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "t.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Mail{}, &models.Attachment{}, &models.MailboxSyncState{}); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	cfg.IMAP.SyncBatchSize = 50

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	client := imapclient.New(conn, &imapclient.Options{})
	if err := client.Login("multi@test", "pass").Wait(); err != nil {
		t.Fatal(err)
	}
	account := &models.MailAccount{Email: "multi@test", Protocol: "imap", SyncMode: "all"}
	ic := &IMAPClient{Client: client, Account: account, conn: conn}

	f := NewFetcher(db, cfg)
	f.folder = "inbox"
	f.DeleteAfterDownload = true
	n, err := f.syncMailbox(ic, "INBOX", "inbox")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("首轮同步新增 %d 封, 删除远程 %v, 跳过 %v", n, f.DeletedRemoteUIDs, f.SkippedDeleteUIDs)

	var cnt int64
	db.Model(&models.Mail{}).Count(&cnt)
	t.Logf("本地邮件总数 = %d (期望 5)", cnt)

	// 服务器上应已无邮件
	sel, err := client.Select("INBOX", &imap.SelectOptions{}).Wait()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("远程 INBOX 剩余 = %d (期望 0)", sel.NumMessages)

	// 追加一封新邮件，再同步一轮，验证不漏信
	u.Append("INBOX", newLiteral([]byte("From: New <n@x.com>\r\nTo: b@x.com\r\nSubject: NewAfter\r\nDate: Mon, 26 Oct 2026 12:00:00 +0800\r\nMessage-ID: <new1@x.com>\r\nContent-Type: text/plain\r\n\r\nnew body\r\n")), &imap.AppendOptions{})
	f2 := NewFetcher(db, cfg)
	f2.folder = "inbox"
	f2.DeleteAfterDownload = true
	n2, err := f2.syncMailbox(ic, "INBOX", "inbox")
	if err != nil {
		t.Fatal(err)
	}
	var subjects []string
	db.Model(&models.Mail{}).Order("id").Pluck("subject", &subjects)
	t.Logf("第二轮新增 %d 封, 删除 %v; 本地 subjects=%v", n2, f2.DeletedRemoteUIDs, subjects)

	var cnt2 int64
	db.Model(&models.Mail{}).Count(&cnt2)
	t.Logf("最终本地总数 = %d (期望 6)", cnt2)
}

// TestSweepBacklog 验证兜底清理：开关开启前同步的存量邮件（增量同步不再抓到）
// 会被 sweepServerMails 删除。
func TestSweepBacklog(t *testing.T) {
	mem := imapmemserver.New()
	u := imapmemserver.NewUser("backlog@test", "pass")
	u.Create("INBOX", nil)
	for i := 1; i <= 3; i++ {
		raw := fmt.Sprintf("From: B%d <b%d@x.com>\r\nTo: x@x.com\r\nSubject: Old%d\r\nDate: Mon, 26 Oct 2026 09:0%d:00 +0800\r\nMessage-ID: <old%d@x.com>\r\nContent-Type: text/plain\r\n\r\noldbody%d\r\n", i, i, i, i%10, i, i)
		if _, err := u.Append("INBOX", newLiteral([]byte(raw)), &imap.AppendOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	mem.AddUser(u)
	srv := imapserver.New(&imapserver.Options{
		NewSession:   func(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) { return mem.NewSession(), nil, nil },
		InsecureAuth: true,
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(ln)
	defer srv.Close()

	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "t.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Mail{}, &models.Attachment{}, &models.MailboxSyncState{}); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	cfg.IMAP.SyncBatchSize = 50

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	client := imapclient.New(conn, &imapclient.Options{})
	if err := client.Login("backlog@test", "pass").Wait(); err != nil {
		t.Fatal(err)
	}
	account := &models.MailAccount{Email: "backlog@test", Protocol: "imap", SyncMode: "all"}
	ic := &IMAPClient{Client: client, Account: account, conn: conn}

	// 模拟"开关未开启时"的同步：邮件入库但远程保留
	f := NewFetcher(db, cfg)
	f.folder = "inbox"
	if _, err := f.syncMailbox(ic, "INBOX", "inbox"); err != nil {
		t.Fatal(err)
	}

	// 现在开启开关，重新同步 → sweep 应删掉 3 封存量邮件
	f2 := NewFetcher(db, cfg)
	f2.folder = "inbox"
	f2.DeleteAfterDownload = true
	if _, err := f2.syncMailbox(ic, "INBOX", "inbox"); err != nil {
		t.Fatal(err)
	}
	swept, err := f2.sweepServerMails(ic)
	if err != nil {
		t.Fatal(err)
	}
	if swept != 3 {
		t.Fatalf("期望兜底清理 3 封存量邮件，实际 %d", swept)
	}
	sel, err := client.Select("INBOX", &imap.SelectOptions{}).Wait()
	if err != nil {
		t.Fatal(err)
	}
	if sel.NumMessages != 0 {
		t.Fatalf("远程 INBOX 应为空，实际 %d 封", sel.NumMessages)
	}
	var cnt int64
	db.Model(&models.Mail{}).Count(&cnt)
	if cnt != 3 {
		t.Fatalf("本地邮件应保留 3 封，实际 %d", cnt)
	}
	t.Log("✅ 兜底清理：开关开启前同步的存量远程邮件已全部删除，本地副本保留")
}
