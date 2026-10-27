// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"novamail/config"
	"novamail/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService 认证服务（多用户模式）
type AuthService struct {
	db        *gorm.DB
	jwtSecret []byte
}

// NewAuthService 创建认证服务实例
func NewAuthService(db *gorm.DB, cfg *config.Config) *AuthService {
	s := &AuthService{
		db:        db,
		jwtSecret: []byte(cfg.Security.JWTSecret),
	}
	s.ensureAtLeastOneAdmin()
	return s
}

// ensureAtLeastOneAdmin 修复历史遗留问题：早期版本注册的用户均无管理员身份，
// 而用户管理功能仅对管理员可见、注册关闭后也无法产生新管理员，形成死锁。
// 启动时若系统中没有任何管理员，则把最早注册的用户提升为管理员。
func (s *AuthService) ensureAtLeastOneAdmin() {
	var adminCount int64
	if err := s.db.Model(&models.User{}).Where("is_admin = ?", true).Count(&adminCount).Error; err != nil {
		log.Printf("[WARN] 检查管理员数量失败: %v", err)
		return
	}
	if adminCount > 0 {
		return
	}
	var first models.User
	if err := s.db.Order("created_at ASC").First(&first).Error; err != nil {
		return // 尚无任何用户，首个注册者会自动成为管理员
	}
	if err := s.db.Model(&first).Update("is_admin", true).Error; err != nil {
		log.Printf("[WARN] 提升管理员失败 (%s): %v", first.Username, err)
		return
	}
	log.Printf("✅ 系统中无管理员，已将最早注册的用户 %s 提升为管理员（请重新登录生效）", first.Username)
}

var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserExists         = errors.New("用户名已被注册")
	ErrRegistrationClosed = errors.New("当前未开放注册，请联系管理员")
	ErrForbidden          = errors.New("无权限执行此操作")
	ErrLastAdmin          = errors.New("不能删除最后一个管理员账号")
)

// HashPassword 对密码进行 bcrypt 哈希
func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// VerifyPassword 验证密码是否匹配哈希值
func (s *AuthService) VerifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GenerateToken 生成 JWT Token（7天有效期）
func (s *AuthService) GenerateToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"is_admin": user.IsAdmin,
		"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ParseToken 解析并验证 JWT Token
func (s *AuthService) ParseToken(tokenStr string) (*jwt.Token, error) {
	return jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("无效的签名方法")
		}
		return s.jwtSecret, nil
	})
}

// registrationEnabled 读取注册开关（默认：尚未有用户时视为开放——首位用户必须能注册）
func (s *AuthService) registrationEnabled() bool {
	var setting models.AppSetting
	if err := s.db.Where("`key` = ?", models.SettingKeyRegistrationEnabled).First(&setting).Error; err != nil {
		return false
	}
	return setting.Value == "true"
}

// SetRegistrationEnabled 设置注册开关（管理员）
func (s *AuthService) SetRegistrationEnabled(enabled bool) error {
	setting := models.AppSetting{Key: models.SettingKeyRegistrationEnabled, Value: boolStr(enabled), UpdatedAt: time.Now()}
	return s.db.Save(&setting).Error
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// GetRegistrationEnabled 查询注册开关
func (s *AuthService) GetRegistrationEnabled() bool {
	return s.registrationEnabled()
}

// Login 用户登录
func (s *AuthService) Login(req models.LoginRequest) (*models.LoginResponse, error) {
	var user models.User
	result := s.db.Where("username = ?", req.Username).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, result.Error
	}

	if err := s.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		return nil, ErrInvalidCredentials
	}
	if user.Disabled {
		return nil, errors.New("账号已被禁用，请联系管理员")
	}

	token, err := s.GenerateToken(&user)
	if err != nil {
		return nil, err
	}

	// 登录成功时兜底：若系统中没有任何管理员，将当前登录用户提升为管理员。
	// 覆盖启动自愈（ensureAtLeastOneAdmin）提升的不是当前登录账号的情况。
	s.ensureAdminFor(&user)

	return &models.LoginResponse{Token: token, Username: user.Username, IsAdmin: user.IsAdmin}, nil
}

// ensureAdminFor 若系统中没有管理员，则把指定用户提升为管理员并更新其 IsAdmin
func (s *AuthService) ensureAdminFor(user *models.User) {
	var adminCount int64
	if err := s.db.Model(&models.User{}).Where("is_admin = ?", true).Count(&adminCount).Error; err != nil {
		return
	}
	if adminCount > 0 {
		return
	}
	if err := s.db.Model(user).Update("is_admin", true).Error; err != nil {
		log.Printf("[WARN] 登录时提升管理员失败 (%s): %v", user.Username, err)
		return
	}
	user.IsAdmin = true
	log.Printf("✅ 系统中无管理员，已将登录用户 %s 提升为管理员", user.Username)
}

// Register 注册新用户（首个用户自动成为管理员；其余用户受注册开关限制）
func (s *AuthService) Register(req models.RegisterRequest) (*models.LoginResponse, error) {
	var count int64
	s.db.Model(&models.User{}).Count(&count)

	// 尚无任何用户：允许注册（系统初始化），成为管理员
	// 已有用户：必须开放注册才允许，且不是管理员
	isFirst := count == 0
	if !isFirst && !s.registrationEnabled() {
		return nil, ErrRegistrationClosed
	}

	hashedPassword, err := s.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username:     req.Username,
		PasswordHash: hashedPassword,
		IsAdmin:      isFirst,
	}
	if err := s.db.Create(user).Error; err != nil {
		// username 唯一索引冲突
		return nil, ErrUserExists
	}

	token, err := s.GenerateToken(user)
	if err != nil {
		return nil, err
	}
	return &models.LoginResponse{Token: token, Username: user.Username, IsAdmin: user.IsAdmin}, nil
}

// GetAuthStatus 查询认证状态（前端据此显示登录/注册入口）
func (s *AuthService) GetAuthStatus() *models.AuthStatusResponse {
	var count int64
	s.db.Model(&models.User{}).Count(&count)

	if count == 0 {
		return &models.AuthStatusResponse{
			SetupRequired:       true,
			RegistrationEnabled: true,
			Message:             "欢迎使用 NovaMail，请先创建管理员账号",
		}
	}
	return &models.AuthStatusResponse{
		SetupRequired:       false,
		RegistrationEnabled: s.registrationEnabled(),
		Message:             "已就绪",
	}
}

// ListUsers 用户列表（管理员）
func (s *AuthService) ListUsers() ([]models.UserListItem, error) {
	var users []models.User
	if err := s.db.Order("created_at ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	items := make([]models.UserListItem, len(users))
	for i, u := range users {
		items[i] = models.UserListItem{ID: u.ID, Username: u.Username, IsAdmin: u.IsAdmin, Disabled: u.Disabled, HasToken: u.ApiToken != "", CreatedAt: u.CreatedAt}
	}
	return items, nil
}

// CreateUser 管理员手动创建用户（不受注册开关限制）
func (s *AuthService) CreateUser(req models.CreateUserRequest) (*models.UserListItem, error) {
	req.Username = strings.TrimSpace(req.Username)
	if utf8.RuneCountInString(req.Username) < 3 || utf8.RuneCountInString(req.Username) > 32 {
		return nil, errors.New("用户名长度应为 3–32 个字符")
	}
	if len(req.Password) < 8 || len(req.Password) > 72 {
		return nil, errors.New("密码长度应为 8–72 字节")
	}
	hashed, err := s.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	user := &models.User{Username: req.Username, PasswordHash: hashed, IsAdmin: req.IsAdmin}
	if err := s.db.Create(user).Error; err != nil {
		return nil, ErrUserExists
	}
	log.Printf("👤 管理员创建了用户 %s", user.Username)
	return &models.UserListItem{ID: user.ID, Username: user.Username, IsAdmin: user.IsAdmin, CreatedAt: user.CreatedAt}, nil
}

// SetUserDisabled 管理员禁用/启用用户（不能禁用自己、不能禁用最后一个管理员）
func (s *AuthService) SetUserDisabled(operatorID, targetID uint, disabled bool) error {
	if operatorID == targetID && disabled {
		return errors.New("不能禁用自己的账号")
	}
	var target models.User
	if err := s.db.First(&target, targetID).Error; err != nil {
		return errors.New("用户不存在")
	}
	if disabled && target.IsAdmin {
		var adminCount int64
		s.db.Model(&models.User{}).Where("is_admin = ? AND disabled = ?", true, false).Count(&adminCount)
		if adminCount <= 1 {
			return ErrLastAdmin
		}
	}
	if err := s.db.Model(&target).Update("disabled", disabled).Error; err != nil {
		return err
	}
	if disabled {
		log.Printf("🚫 管理员禁用了用户 %s", target.Username)
	} else {
		log.Printf("✅ 管理员启用了用户 %s", target.Username)
	}
	return nil
}

// DeleteUser 删除用户（管理员，不能删自己，不能删最后一个管理员）
func (s *AuthService) DeleteUser(operatorID, targetID uint) error {
	if operatorID == targetID {
		return errors.New("不能删除自己的账号")
	}
	var target models.User
	if err := s.db.First(&target, targetID).Error; err != nil {
		return err
	}
	if target.IsAdmin {
		var adminCount int64
		s.db.Model(&models.User{}).Where("is_admin = ?", true).Count(&adminCount)
		if adminCount <= 1 {
			return ErrLastAdmin
		}
	}

	// 停掉该用户所有邮箱账号的同步 Worker
	var accountIDs []uint
	s.db.Model(&models.MailAccount{}).Where("user_id = ?", targetID).Pluck("id", &accountIDs)
	_ = accountIDs

	// 数据隔离：级联删除该用户的邮箱账号、邮件、附件、草稿、联系人、推送订阅
	return s.deleteUserData(targetID)
}

// deleteUserData 级联删除用户全部业务数据
func (s *AuthService) deleteUserData(userID uint) error {
	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 该用户所有邮箱账号 ID
	var accountIDs []uint
	tx.Model(&models.MailAccount{}).Where("user_id = ?", userID).Pluck("id", &accountIDs)

	if len(accountIDs) > 0 {
		var mailIDs []uint
		tx.Model(&models.Mail{}).Where("account_id IN ?", accountIDs).Pluck("id", &mailIDs)
		if len(mailIDs) > 0 {
			tx.Where("mail_id IN ?", mailIDs).Delete(&models.Attachment{})
		}
		tx.Where("account_id IN ?", accountIDs).Delete(&models.Mail{})
		tx.Where("account_id IN ?", accountIDs).Delete(&models.MailboxSyncState{})
		tx.Where("id IN ?", accountIDs).Delete(&models.MailAccount{})
	}

	tx.Where("user_id = ?", userID).Delete(&models.Draft{})
	tx.Where("user_id = ?", userID).Delete(&models.Contact{})
	tx.Where("user_id = ?", userID).Delete(&models.PushSubscription{})
	tx.Where("user_id = ?", userID).Delete(&models.Webhook{})
	tx.Where("id = ?", userID).Delete(&models.User{})
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

// ChangePassword 修改密码
func (s *AuthService) ChangePassword(userID uint, req models.ChangePasswordRequest) error {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}
	if err := s.VerifyPassword(req.OldPassword, user.PasswordHash); err != nil {
		return errors.New("原密码错误")
	}
	hashed, err := s.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	return s.db.Model(&user).Update("password_hash", hashed).Error
}

// Me 返回当前登录用户的实时信息（含数据库中的 is_admin，避免旧 JWT 里的过期身份）
func (s *AuthService) Me(userID uint) (*models.UserListItem, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &models.UserListItem{ID: user.ID, Username: user.Username, IsAdmin: user.IsAdmin, CreatedAt: user.CreatedAt}, nil
}

// SeedDefaultUser 开发环境：自动创建默认管理员（仅在无用户时生效）
func (s *AuthService) SeedDefaultUser(username, password string) error {
	var count int64
	s.db.Model(&models.User{}).Count(&count)
	if count > 0 {
		return nil
	}

	hashedPassword, err := s.HashPassword(password)
	if err != nil {
		return err
	}

	user := &models.User{
		Username:     username,
		PasswordHash: hashedPassword,
		IsAdmin:      true,
	}

	err = s.db.Create(user).Error
	if err != nil {
		return err
	}

	log.Printf("✅ 已创建默认管理员账号: %s", username)
	return nil
}

// _ = fmt 用于开发日志输出
var _ = fmt.Sprintf

// ---- API Token（外部 AI / 脚本读取邮件用）----

// generateApiToken 生成 32 字节随机 hex token
func generateApiToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// EnsureApiToken 获取当前用户的 API Token；无则生成
func (s *AuthService) EnsureApiToken(userID uint) (string, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return "", errors.New("用户不存在")
	}
	if user.Disabled {
		return "", errors.New("账号已被禁用")
	}
	if user.ApiToken != "" {
		return user.ApiToken, nil
	}
	tok, err := generateApiToken()
	if err != nil {
		return "", err
	}
	if err := s.db.Model(&user).Update("api_token", tok).Error; err != nil {
		return "", err
	}
	return tok, nil
}

// RevokeApiToken 吊销当前用户的 API Token
func (s *AuthService) RevokeApiToken(userID uint) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("api_token", "").Error
}

// UserByApiToken 按 API Token 查用户（外部 API 认证用；禁用用户视为无效）
func (s *AuthService) UserByApiToken(token string) (*models.User, error) {
	if len(token) < 32 {
		return nil, errors.New("无效的 API Token")
	}
	var user models.User
	if err := s.db.Where("api_token = ?", token).First(&user).Error; err != nil {
		return nil, errors.New("无效的 API Token")
	}
	if user.Disabled {
		return nil, errors.New("账号已被禁用")
	}
	return &user, nil
}
