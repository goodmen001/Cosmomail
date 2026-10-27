// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package services

import (
	"fmt"
	"log"
	"strings"
	"time"

	"cosmomail/config"
	"cosmomail/contacts"
	"cosmomail/imap"
	"cosmomail/models"
	pop3pkg "cosmomail/pop3"
	"cosmomail/smtp"

	"gorm.io/gorm"
)

// MailService 邮件业务逻辑
type MailService struct {
	db     *gorm.DB
	config *config.Config
}

// NewMailService 创建邮件 Service 实例
func NewMailService(db *gorm.DB, cfg *config.Config) *MailService {
	return &MailService{db: db, config: cfg}
}

// List 获取当前用户的邮件列表（分页、搜索、筛选；按账号归属隔离）
func (s *MailService) List(userID uint, filter models.MailListFilter) ([]models.MailListItem, int64, error) {
	var mails []models.Mail
	var total int64

	// 默认参数
	if filter.Page < 1 {
		filter.Page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := s.db.Model(&models.Mail{}).
		Joins("LEFT JOIN mail_accounts ON mails.account_id = mail_accounts.id").
		Where("mail_accounts.user_id = ?", userID)

	// 筛选：指定邮箱账号
	if filter.AccountID > 0 {
		query = query.Where("mails.account_id = ?", filter.AccountID)
	}

	// 筛选：文件夹（inbox/sent）
	if filter.Folder != "" {
		query = query.Where("mails.folder = ?", filter.Folder)
	}

	// 筛选：已读/未读
	if filter.IsRead != nil {
		query = query.Where("mails.is_read = ?", *filter.IsRead)
	}

	// 筛选：有/无附件
	if filter.HasAttachment != nil {
		query = query.Where("mails.has_attachment = ?", *filter.HasAttachment)
	}

	// 搜索：发件人或主题关键词
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		query = query.Where("mails.from LIKE ? OR mails.subject LIKE ?", keyword, keyword)
	}

	// 统计总数（同样按用户隔离）
	countQuery := s.db.Model(&models.Mail{}).
		Joins("LEFT JOIN mail_accounts ON mails.account_id = mail_accounts.id").
		Where("mail_accounts.user_id = ?", userID)
	if filter.AccountID > 0 {
		countQuery = countQuery.Where("account_id = ?", filter.AccountID)
	}
	if filter.Folder != "" {
		countQuery = countQuery.Where("folder = ?", filter.Folder)
	}
	if filter.IsRead != nil {
		countQuery = countQuery.Where("is_read = ?", *filter.IsRead)
	}
	if filter.HasAttachment != nil {
		countQuery = countQuery.Where("has_attachment = ?", *filter.HasAttachment)
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		countQuery = countQuery.Where("`from` LIKE ? OR subject LIKE ?", keyword, keyword)
	}
	countQuery.Count(&total)

	// 排序
	sortBy := "sent_at"
	if filter.SortBy != "" {
		sortBy = strings.ToLower(filter.SortBy)
	}
	switch sortBy {
	case "created_at":
		sortBy = "mails.created_at"
	case "from", "subject":
		sortBy = "mails." + sortBy
	default:
		sortBy = "mails.sent_at"
	}

	sortOrder := "DESC"
	if strings.ToUpper(filter.SortOrder) == "ASC" {
		sortOrder = "ASC"
	}

	// 分页查询（只取必要字段，不含正文）
	offset := (filter.Page - 1) * pageSize
	fields := "mails.id, mails.account_id, mail_accounts.name AS account_name," +
		" mails.folder, mails.`from`, mails.`to`, mails.subject, mails.text_body," +
		" mails.sent_at, mails.is_read, mails.is_starred, mails.has_attachment," +
		" mails.size, mails.created_at"

	err := query.Select(fields).
		Offset(offset).Limit(pageSize).
		Order(sortBy + " " + sortOrder).
		Scan(&mails).Error

	if err != nil {
		return nil, 0, err
	}

	// 构建预览文本
	items := make([]models.MailListItem, len(mails))
	for i, m := range mails {
		item := models.MailListItem{
			ID:            m.ID,
			AccountID:     m.AccountID,
			Folder:        m.Folder,
			From:          m.From,
			To:            m.To,
			Subject:       m.Subject,
			SentAt:        m.SentAt,
			IsRead:        m.IsRead,
			IsStarred:     m.IsStarred,
			HasAttachment: m.HasAttachment,
			Size:          m.Size,
			CreatedAt:     m.CreatedAt,
		}
		// 预览文本（优先纯文本，其次 HTML 去标签）
		if m.TextBody.Valid {
			item.Preview = truncatePreview(m.TextBody.String, 150)
		} else if m.HTMLBody.Valid {
			item.Preview = stripHTML(m.HTMLBody.String)
			item.Preview = truncatePreview(item.Preview, 150)
		}
		items[i] = item
	}

	return items, total, nil
}

// GetByID 获取邮件详情（含正文和附件，限定当前用户）
func (s *MailService) GetByID(userID, id uint) (*models.MailResponse, error) {
	var mail models.Mail

	if err := s.db.Preload("Attachments").Preload("Account").
		Joins("LEFT JOIN mail_accounts ON mails.account_id = mail_accounts.id").
		Where("mail_accounts.user_id = ?", userID).
		First(&mail, id).Error; err != nil {
		return nil, err
	}

	resp := models.MailResponse{
		ID:            mail.ID,
		AccountID:     mail.AccountID,
		Folder:        mail.Folder,
		MessageID:     mail.MessageID,
		From:          mail.From,
		To:            mail.To,
		Cc:            mail.Cc,
		Subject:       mail.Subject,
		SentAt:        mail.SentAt,
		IsRead:        mail.IsRead,
		IsStarred:     mail.IsStarred,
		HasAttachment: mail.HasAttachment,
		Size:          mail.Size,
		CreatedAt:     mail.CreatedAt,
	}

	if mail.Account != nil {
		resp.AccountName = mail.Account.Name
	}

	if mail.TextBody.Valid {
		resp.TextBody = &mail.TextBody.String
	}
	if mail.HTMLBody.Valid {
		resp.HTMLBody = &mail.HTMLBody.String
	}

	// ⭐ 调试日志：记录邮件详情获取情况
	log.Printf("🔍 [DEBUG] GetByID (id=%d): text_valid=%v, text_len=%d, html_valid=%v, html_len=%d, attachments=%d, has_attachment=%v",
		id, mail.TextBody.Valid, len(mail.TextBody.String), mail.HTMLBody.Valid, len(mail.HTMLBody.String),
		len(mail.Attachments), mail.HasAttachment)

	// 转换附件响应
	resp.Attachments = make([]models.AttachmentResp, len(mail.Attachments))
	for i, att := range mail.Attachments {
		resp.Attachments[i] = models.AttachmentResp{
			ID:          att.ID,
			MailID:      att.MailID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
			SizeHuman:   formatSize(att.Size),
			IsCached:    att.IsCached,
			CreatedAt:   att.CreatedAt,
		}
	}

	return &resp, nil
}

// MarkAsRead 标记邮件为已读/未读
func (s *MailService) MarkAsRead(userID, id uint, isRead bool) error {
	return s.db.Model(&models.Mail{}).
		Joins("LEFT JOIN mail_accounts ON mails.account_id = mail_accounts.id").
		Where("mails.id = ? AND mail_accounts.user_id = ?", id, userID).
		Update("is_read", isRead).Error
}

// BatchMarkAsReadResult 批量标记已读/未读操作结果
type BatchMarkAsReadResult struct {
	Success bool   `json:"success"`
	Updated int64  `json:"updated"`
	Failed  []uint `json:"failed"`
}

// BatchMarkAsRead 批量标记邮件已读/未读状态
func (s *MailService) BatchMarkAsRead(userID uint, ids []uint, isRead bool) *BatchMarkAsReadResult {
	result := &BatchMarkAsReadResult{}

	// 批量更新数据库（使用 IN 查询，一次 SQL 完成；限定当前用户的账号）
	dbResult := s.db.Model(&models.Mail{}).
		Joins("LEFT JOIN mail_accounts ON mails.account_id = mail_accounts.id").
		Where("mails.id IN ? AND mail_accounts.user_id = ?", ids, userID).
		Update("is_read", isRead)

	if dbResult.Error != nil {
		result.Failed = ids
		return result
	}

	result.Updated = dbResult.RowsAffected
	result.Success = true
	return result
}

// MarkAllAsRead 将符合筛选条件的所有邮件标记为已读
func (s *MailService) MarkAllAsRead(userID uint, filter models.MailListFilter) (int64, error) {
	query := s.db.Model(&models.Mail{}).
		Joins("LEFT JOIN mail_accounts ON mails.account_id = mail_accounts.id").
		Where("mail_accounts.user_id = ?", userID)

	if filter.AccountID > 0 {
		query = query.Where("account_id = ?", filter.AccountID)
	}
	if filter.Folder != "" {
		query = query.Where("folder = ?", filter.Folder)
	}
	if filter.IsRead != nil {
		query = query.Where("is_read = ?", *filter.IsRead)
	}
	if filter.HasAttachment != nil {
		query = query.Where("has_attachment = ?", *filter.HasAttachment)
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		query = query.Where("`from` LIKE ? OR subject LIKE ?", keyword, keyword)
	}

	result := query.Update("is_read", true)
	return result.RowsAffected, result.Error
}

// MarkAsStarred 标记邮件星标
func (s *MailService) MarkAsStarred(userID, id uint, starred bool) error {
	return s.db.Model(&models.Mail{}).
		Joins("LEFT JOIN mail_accounts ON mails.account_id = mail_accounts.id").
		Where("mails.id = ? AND mail_accounts.user_id = ?", id, userID).
		Update("is_starred", starred).Error
}

// DeleteResult 删除操作结果
type DeleteResult struct {
	Success           bool   `json:"success"`
	DeletedFromServer bool   `json:"deleted_from_server"`           // 是否成功从源服务器删除
	ServerDeleteError string `json:"server_delete_error,omitempty"` // 源服务器删除错误信息（如有）
}

// Delete 删除邮件及其附件，可选同步删除源服务器上的邮件
func (s *MailService) Delete(ownerID, id uint) *DeleteResult {
	result := &DeleteResult{Success: false}

	// 先获取邮件信息（需要 account_id 和 message_uid，限定当前用户）
	var mail models.Mail
	if err := s.db.Joins("LEFT JOIN mail_accounts ON mails.account_id = mail_accounts.id").
		Where("mail_accounts.user_id = ?", ownerID).
		First(&mail, id).Error; err != nil {
		return result // 邮件不存在
	}

	// 查询账号配置
	var account models.MailAccount
	if err := s.db.First(&account, mail.AccountID).Error; err != nil {
		return result // 邮箱账号不存在
	}

	// 如果开启了源服务器删除，先删除远程邮件
	if account.DeleteOnServer && s.config != nil {
		if err := s.deleteFromServer(&account, &mail); err != nil {
			log.Printf("[WARN] 源服务器删除失败 (mail_id=%d): %v", id, err)
			result.ServerDeleteError = err.Error()
			// 远程删除失败不阻止本地删除，仅记录错误
		} else {
			result.DeletedFromServer = true
		}
	}

	tx := s.db.Begin()

	// 删除附件
	tx.Where("mail_id = ?", id).Delete(&models.Attachment{})

	// 删除邮件
	if dbErr := tx.Delete(&models.Mail{}, id).Error; dbErr != nil {
		tx.Rollback()
		return result
	}

	if commitErr := tx.Commit().Error; commitErr != nil {
		return result
	}

	result.Success = true
	return result
}

// BatchDeleteResult 批量删除操作结果
type BatchDeleteResult struct {
	Success          bool                   `json:"success"`
	Deleted          int64                  `json:"deleted"`
	Failed           []uint                 `json:"failed"`
	ServerSyncResult map[uint]*DeleteResult `json:"server_sync_result,omitempty"` // 每封邮件的同步删除状态
}

// BatchDelete 批量删除邮件
func (s *MailService) BatchDelete(userID uint, ids []uint) *BatchDeleteResult {
	result := &BatchDeleteResult{
		ServerSyncResult: make(map[uint]*DeleteResult),
	}

	for _, id := range ids {
		delResult := s.Delete(userID, id)
		result.ServerSyncResult[id] = delResult

		if delResult.Success {
			result.Deleted++
		} else {
			result.Failed = append(result.Failed, id)
			log.Printf("[MailService] 批量删除失败 (id=%d)", id)
		}
	}

	result.Success = len(result.Failed) == 0
	return result
}

// deleteFromServer 在源服务器上删除对应邮件
func (s *MailService) deleteFromServer(account *models.MailAccount, mail *models.Mail) error {
	client, err := imap.NewMailClient(account, s.config)
	if err != nil {
		return fmt.Errorf("创建客户端失败: %w", err)
	}
	defer client.Close()

	if err := client.Authenticate(); err != nil {
		return fmt.Errorf("认证失败: %w", err)
	}

	switch account.Protocol {
	case "pop3", "pop3-no-ssl":
		pc, ok := client.(*pop3pkg.POP3Client)
		if !ok {
			return fmt.Errorf("类型断言失败")
		}
		// POP3 使用序号（message_uid 存储的即 POP3 序号）
		return pc.DeleteMessage(int(mail.MessageUID))
	default:
		ic, ok := client.(*imap.IMAPClient)
		if !ok {
			return fmt.Errorf("类型断言失败")
		}
		return ic.DeleteMessage(mail.MessageUID)
	}
}

// GetStats 获取邮件统计信息（限定当前用户的账号）
func (s *MailService) GetStats(ownerID uint, accountID *uint) map[string]int64 {
	stats := make(map[string]int64)

	userFilter := func(q *gorm.DB) *gorm.DB {
		q = q.Joins("LEFT JOIN mail_accounts ON mails.account_id = mail_accounts.id").
			Where("mail_accounts.user_id = ?", ownerID)
		if accountID != nil {
			q = q.Where("mails.account_id = ?", *accountID)
		}
		return q
	}

	query := userFilter(s.db.Model(&models.Mail{}))

	var total, unread, withAttachment, starred, sent int64

	// 总数
	query.Count(&total)
	stats["total"] = total

	// 已发送
	userFilter(s.db.Model(&models.Mail{})).Where("mails.folder = 'sent'").Count(&sent)
	stats["sent"] = sent

	// 未读数
	userFilter(s.db.Model(&models.Mail{})).Where("mails.is_read = false").Count(&unread)
	stats["unread"] = unread

	// 有附件
	userFilter(s.db.Model(&models.Mail{})).Where("mails.has_attachment = true").Count(&withAttachment)
	stats["with_attachment"] = withAttachment

	// 已标星
	userFilter(s.db.Model(&models.Mail{})).Where("mails.is_starred = true").Count(&starred)
	stats["starred"] = starred

	return stats
}

// truncatePreview 截取预览文本
func truncatePreview(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if len([]rune(text)) <= maxLen {
		return text
	}
	runes := []rune(text)
	return string(runes[:maxLen]) + "..."
}

// formatSize 格式化文件大小为人类可读格式
func formatSize(size int64) string {
	if size < 0 {
		return "未知"
	}
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// stripHTML 移除 HTML 标签，返回纯文本
// 会完整移除 <style> 和 <script> 标签及其内部内容，避免 CSS/JS 代码泄漏
func stripHTML(html string) string {
	s := html

	// 1. 先移除 <style>...</style> 和 <script>...</script> 块（大小写不敏感）
	for _, tag := range []string{"style", "script"} {
		for {
			lower := strings.ToLower(s)
			startTag := "<" + tag
			endTag := "</" + tag + ">"

			startIdx := strings.Index(lower, startTag)
			if startIdx == -1 {
				break
			}
			startClose := strings.Index(s[startIdx:], ">")
			if startClose == -1 {
				break
			}
			contentStart := startIdx + startClose + 1
			endIdx := strings.Index(strings.ToLower(s[contentStart:]), endTag)
			if endIdx == -1 {
				s = s[:startIdx]
				break
			}
			s = s[:startIdx] + s[contentStart+endIdx+len(endTag):]
		}
	}

	// 2. 移除剩余 HTML 标签
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}

	// 3. 清理多余空白
	raw := result.String()
	var cleaned strings.Builder
	prevSpace := false
	for _, r := range raw {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevSpace {
				cleaned.WriteRune(' ')
				prevSpace = true
			}
		} else {
			cleaned.WriteRune(r)
			prevSpace = false
		}
	}
	return strings.TrimSpace(cleaned.String())
}

// SendMail 发送邮件（限定当前用户的账号；成功后记录/去重收件人）
func (s *MailService) SendMail(ownerID uint, req smtp.SendRequest) (*smtp.SendResult, error) {
	var account models.MailAccount
	if err := s.db.Where("id = ? AND user_id = ?", req.AccountID, ownerID).First(&account).Error; err != nil {
		return nil, fmt.Errorf("邮箱账号不存在 (ID=%d)", req.AccountID)
	}

	res, err := smtp.SendMail(&account, &req, s.db)
	if err != nil {
		return res, err
	}

	// 记录收件人（去重：按 user_id + email 唯一）
	s.recordContacts(ownerID, req.AccountID, req.To)
	return res, nil
}

// recordContacts 记录发信收件人，存在则累加次数并刷新时间
func (s *MailService) recordContacts(ownerID, accountID uint, emails []string) {
	now := time.Now()
	seen := make(map[string]bool)
	for _, raw := range emails {
		email := models.NormalizeEmail(raw)
		if email == "" || !strings.Contains(email, "@") || seen[email] {
			continue
		}
		seen[email] = true

		var contact models.Contact
		err := s.db.Where("user_id = ? AND email = ?", ownerID, email).First(&contact).Error
		if err != nil {
			contact = models.Contact{
				UserID: ownerID, Email: email, AccountID: accountID,
				LastUsedAt: now, UseCount: 1,
			}
			if err := s.db.Create(&contact).Error; err != nil {
				log.Printf("[WARN] 保存联系人失败: %v", err)
			}
			continue
		}
		s.db.Model(&contact).Updates(map[string]interface{}{
			"last_used_at": now,
			"use_count":    contact.UseCount + 1,
		})
	}
}

// ListContacts 获取当前用户的联系人列表（按最近使用排序）
// accountID > 0 时只返回该邮箱账号的通讯录；keyword 同时匹配邮箱和姓名
func (s *MailService) ListContacts(ownerID, accountID uint, keyword string, limit int) ([]models.Contact, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	query := s.db.Where("user_id = ?", ownerID)
	if accountID > 0 {
		query = query.Where("account_id = ?", accountID)
	}
	if keyword != "" {
		kw := "%" + models.NormalizeEmail(keyword) + "%"
		query = query.Where("(email LIKE ? OR name LIKE ?)", kw, kw)
	}
	var contacts []models.Contact
	err := query.Order("last_used_at DESC").Limit(limit).Find(&contacts).Error
	return contacts, err
}

// SyncAccountContacts 全量提取指定账号已同步邮件中的联系人
func (s *MailService) SyncAccountContacts(ownerID, accountID uint) (int64, error) {
	var acct models.MailAccount
	if err := s.db.Where("id = ? AND user_id = ?", accountID, ownerID).First(&acct).Error; err != nil {
		return 0, err
	}
	return contacts.NewSyncService(s.db).SyncAccountContacts(ownerID, accountID, acct.Email)
}

// SyncAccountCardDAV 从指定账号配置的 CardDAV 服务器同步联系人
func (s *MailService) SyncAccountCardDAV(ownerID, accountID uint) (int, error) {
	var acct models.MailAccount
	if err := s.db.Where("id = ? AND user_id = ?", accountID, ownerID).First(&acct).Error; err != nil {
		return 0, err
	}
	if !acct.CardDAVEnabled || acct.CardDAVURL == "" {
		return 0, fmt.Errorf("该账号未启用 CardDAV 或未配置地址簿 URL")
	}
	cfg := contacts.CardDAVConfig{
		Enabled:  acct.CardDAVEnabled,
		URL:      acct.CardDAVURL,
		Username: acct.CardDAVUsername,
		Password: acct.CardDAVPassword,
	}
	return contacts.NewSyncService(s.db).SyncCardDAV(ownerID, accountID, cfg)
}

// SaveSentMail 保存已发送邮件到数据库
func (s *MailService) SaveSentMail(mail *models.Mail) error {
	return s.db.Create(mail).Error
}

// GetAccountEmail 获取指定账号的邮箱地址
func (s *MailService) GetAccountEmail(accountID uint) (string, error) {
	var account models.MailAccount
	if err := s.db.Select("email").First(&account, accountID).Error; err != nil {
		return "", err
	}
	return account.Email, nil
}
