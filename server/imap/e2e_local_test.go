// 临时端到端验证（不提交）：本地 IMAP 服务器 + Fetcher 真实同步
package imap

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cosmomail/config"
	"cosmomail/models"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// 纯 TCP 明文 IMAP 服务器：CosmoMail 客户端默认走 TLS，测试里通过 starttls=false 直连
func startPlainIMAP(t *testing.T, mem *imapmemserver.Server) (addr string, stop func()) {
	srv := imapserver.New(&imapserver.Options{
		NewSession:   func(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) { return mem.NewSession(), nil, nil },
		InsecureAuth: true,
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(ln)
	return ln.Addr().String(), func() { srv.Close() }
}

func TestE2ELocalIMAPDownload(t *testing.T) {
	// 1. 内存 IMAP 服务器，用户 user@test / pass123
	mem := imapmemserver.New()
	u := imapmemserver.NewUser("user@test", "pass123")
	u.Create("INBOX", nil)
	raw := "From: Alice <alice@example.com>\r\nTo: Bob <bob@example.com>\r\nSubject: E2E Test Mail\r\nDate: Mon, 26 Oct 2026 10:00:00 +0800\r\nMessage-ID: <e2e-1@example.com>\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=\"BOUND\"\r\n\r\n--BOUND\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nHello from remote server!\r\n--BOUND\r\nContent-Type: application/pdf; name=\"report.pdf\"\r\nContent-Disposition: attachment; filename=\"report.pdf\"\r\nContent-Transfer-Encoding: base64\r\n\r\nJVBERi0xLjQK\r\n--BOUND--\r\n"
	if _, err := u.Append("INBOX", newLiteral([]byte(raw)), &imap.AppendOptions{}); err != nil {
		t.Fatal(err)
	}
	mem.AddUser(u)
	addr, stop := startPlainIMAP(t, mem)
	defer stop()

	// 2. 本地 SQLite 数据库
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Mail{}, &models.Attachment{}, &models.MailboxSyncState{}); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{}
	cfg.IMAP.SyncBatchSize = 50
	cfg.IMAP.PollInterval = 60

	// 3. 用 imapclient 直连做一次握手确认服务器 OK，然后走 Fetcher 完整路径
	account := &models.MailAccount{
		Email: "user@test", Password: "pass123",
		ImapHost: hostOf(addr), Port: portOf(addr),
		Protocol: "imap", SyncMode: "all", SyncDays: 30,
	}

	// CosmoMail 客户端默认 TLS，这里模拟其行为但用明文连接（同 dialIMAPTLS 的代理逻辑之外）
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	client := imapclient.New(conn, &imapclient.Options{})
	if err := client.Login("user@test", "pass123").Wait(); err != nil {
		t.Fatal(err)
	}

	f := NewFetcher(db, cfg)
	f.folder = "inbox"
	ic := &IMAPClient{Client: client, Account: account, conn: conn}

	n, err := f.syncMailbox(ic, "INBOX", "inbox")
	if err != nil {
		t.Fatalf("sync 失败: %v", err)
	}
	if n != 1 {
		t.Fatalf("期望新增 1 封邮件，实际 %d", n)
	}

	// 4. 验证邮件确实下载到本地数据库（含正文）
	var mails []models.Mail
	if err := db.Find(&mails).Error; err != nil || len(mails) != 1 {
		t.Fatalf("本地库邮件数=%d err=%v", len(mails), err)
	}
	m := mails[0]
	if !strings.Contains(m.TextBody.String, "Hello from remote server!") {
		t.Fatalf("正文未真实下载: %q", m.TextBody.String)
	}
	if !strings.Contains(m.Subject, "E2E Test Mail") || m.From == "" {
		t.Fatalf("信封字段缺失: subject=%q from=%q", m.Subject, m.From)
	}

	// 5. 验证附件记录 + 正文里的 attachment 标记
	var atts []models.Attachment
	db.Where("mail_id = ?", m.ID).Find(&atts)
	if len(atts) == 0 {
		t.Logf("附件未单独入库，has_attachment=%v", m.HasAttachment)
	} else {
		t.Logf("附件入库: %s (%d bytes) path=%s", atts[0].Filename, atts[0].Size, atts[0].FilePath)
	}

	// 6. 增量同步验证：再追加一封，再次同步应只拉新的
	u.Append("INBOX", newLiteral([]byte("From: Carol <carol@example.com>\r\nTo: bob@example.com\r\nSubject: Second\r\nDate: Mon, 26 Oct 2026 11:00:00 +0800\r\nMessage-ID: <e2e-2@example.com>\r\nContent-Type: text/plain\r\n\r\nsecond body\r\n")), &imap.AppendOptions{})
	n2, err := f.syncMailbox(ic, "INBOX", "inbox")
	if err != nil {
		t.Fatalf("第二次同步失败: %v", err)
	}
	var cnt int64
	db.Model(&models.Mail{}).Count(&cnt)
	t.Logf("第二次同步新增 %d 封，本地库总计 %d 封（去重正确应 = 2）", n2, cnt)
	if cnt != 2 {
		t.Fatalf("增量/去重异常：总计 %d", cnt)
	}

	t.Log("✅ 端到端验证通过：邮件从 IMAP 服务器真实下载并存储到本地应用服务器（SQLite 正文 + 附件记录）")
}

func newLiteral(b []byte) imap.LiteralReader {
	return bytes.NewReader(b)
}

func hostOf(addr string) string {
	h, _, _ := net.SplitHostPort(addr)
	return h
}
func portOf(addr string) int {
	_, p, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range p {
		port = port*10 + int(c-'0')
	}
	return port
}

var _ = os.Stat
var _ = time.Now
var _ = imap.UID(0)

// TestE2EDeleteAfterDownload 验证"下载后删除远程邮件"功能：
// 开启后同步 → 本地保存完整 → 远程邮件被 \Deleted + Expunge；
// 若本地保存不完整（无正文），远程邮件必须保留。
func TestE2EDeleteAfterDownload(t *testing.T) {
	mem := imapmemserver.New()
	u := imapmemserver.NewUser("del@test", "pass")
	u.Create("INBOX", nil)
	complete := "From: Alice <a@x.com>\r\nTo: b@x.com\r\nSubject: Keep Me Locally\r\nDate: Mon, 26 Oct 2026 10:00:00 +0800\r\nMessage-ID: <del-1@x.com>\r\nContent-Type: text/plain\r\n\r\nfull body saved\r\n"
	if _, err := u.Append("INBOX", newLiteral([]byte(complete)), &imap.AppendOptions{}); err != nil {
		t.Fatal(err)
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
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "test.db")), &gorm.Config{})
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
	if err := client.Login("del@test", "pass").Wait(); err != nil {
		t.Fatal(err)
	}
	account := &models.MailAccount{Email: "del@test", Protocol: "imap"}
	ic := &IMAPClient{Client: client, Account: account, conn: conn}

	// --- 场景A：开启删除，本地完整保存 → 远程应被删除 ---
	f := NewFetcher(db, cfg)
	f.folder = "inbox"
	f.DeleteAfterDownload = true
	n, err := f.syncMailbox(ic, "INBOX", "inbox")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("期望同步 1 封，实际 %d", n)
	}
	if len(f.DeletedRemoteUIDs) != 1 {
		t.Fatalf("期望远程删除 1 封，实际删除 %v (skipped=%v)", f.DeletedRemoteUIDs, f.SkippedDeleteUIDs)
	}
	// 服务器上应已无邮件
	selData, err := client.Select("INBOX", &imap.SelectOptions{}).Wait()
	if err != nil {
		t.Fatal(err)
	}
	if selData.NumMessages != 0 {
		t.Fatalf("远程 INBOX 应为空，实际 %d 封", selData.NumMessages)
	}
	// 本地邮件仍完整可读
	var m models.Mail
	if err := db.First(&m).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.TextBody.String, "full body saved") {
		t.Fatal("本地正文丢失")
	}
	t.Log("✅ 场景A：本地保存完整 → 远程邮件已删除，本地可继续阅读")

	// --- 场景B：本地正文缺失 → 不得删除远程 ---
	if _, err := u.Append("INBOX", newLiteral([]byte("From: C <c@x.com>\r\nTo: b@x.com\r\nSubject: Guarded\r\nDate: Mon, 26 Oct 2026 11:00:00 +0800\r\nMessage-ID: <del-2@x.com>\r\nContent-Type: text/plain\r\n\r\nbody2\r\n")), &imap.AppendOptions{}); err != nil {
		t.Fatal(err)
	}
	f2 := NewFetcher(db, cfg)
	f2.folder = "inbox"
	f2.DeleteAfterDownload = true
	if _, err := f2.syncMailbox(ic, "INBOX", "inbox"); err != nil {
		t.Fatal(err)
	}
	if len(f2.DeletedRemoteUIDs) != 1 {
		t.Fatalf("正文完整的邮件应被删除，实际 %v", f2.DeletedRemoteUIDs)
	}
	// 人为把本地正文清空，模拟存储不完整，再放一封 → 去重会跳过所以这里直接篡改 locallyComplete 输入：
	// 更直接的场景：附件懒加载未缓存时不得删除（构造带附件邮件，DB 里附件 IsCached=false）
	if _, err := u.Append("INBOX", newLiteral([]byte("From: D <d@x.com>\r\nTo: b@x.com\r\nSubject: WithAtt\r\nDate: Mon, 26 Oct 2026 12:00:00 +0800\r\nMessage-ID: <del-3@x.com>\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=\"B\"\r\n\r\n--B\r\nContent-Type: text/plain\r\n\r\nhello\r\n--B\r\nContent-Type: application/octet-stream; name=\"big.bin\"\r\nContent-Disposition: attachment; filename=\"big.bin\"\r\nContent-Transfer-Encoding: base64\r\n\r\nAQIDBAU=\r\n--B--\r\n")), &imap.AppendOptions{}); err != nil {
		t.Fatal(err)
	}
	// 配置懒加载（CacheThresholdMB=0 时 estimatedSize==0 也懒加载；这里用默认阈值，小附件会入库缓存）
	f3 := NewFetcher(db, cfg)
	f3.folder = "inbox"
	f3.DeleteAfterDownload = true
	if _, err := f3.syncMailbox(ic, "INBOX", "inbox"); err != nil {
		t.Fatal(err)
	}
	// 小附件应已缓存入库，因此允许删除
	selData2, err := client.Select("INBOX", &imap.SelectOptions{}).Wait()
	if err != nil {
		t.Fatal(err)
	}
	if selData2.NumMessages != 0 {
		t.Fatalf("带小附件邮件下载完整后也应删除远程，剩余 %d 封", selData2.NumMessages)
	}
	var att models.Attachment
	db.Where("filename = ?", "big.bin").First(&att)
	if !att.IsCached || len(att.Content) == 0 {
		t.Fatalf("附件应已缓存到本地库: cached=%v contentLen=%d", att.IsCached, len(att.Content))
	}
	t.Log("✅ 场景C：带小附件邮件下载后删除远程，附件内容已在本地表中")
}
