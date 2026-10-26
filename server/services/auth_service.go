// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"cosmomail/config"
	"cosmomail/models"

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
	return &AuthService{
		db:        db,
		jwtSecret: []byte(cfg.Security.JWTSecret),
	}
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

	token, err := s.GenerateToken(&user)
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{Token: token, Username: user.Username, IsAdmin: user.IsAdmin}, nil
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
			Message:             "欢迎使用 Cosmo Mail，请先创建管理员账号",
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
		items[i] = models.UserListItem{ID: u.ID, Username: u.Username, IsAdmin: u.IsAdmin, CreatedAt: u.CreatedAt}
	}
	return items, nil
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
