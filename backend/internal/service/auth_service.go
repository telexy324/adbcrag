package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ops-kb-rag/backend/internal/config"
	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/model"
	"ops-kb-rag/backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	cfg   *config.Config
	users *repository.UserRepository
}

type Claims struct {
	UserID   uint64 `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Exp      int64  `json:"exp"`
}

func NewAuthService(cfg *config.Config, users *repository.UserRepository) *AuthService {
	return &AuthService{cfg: cfg, users: users}
}

func (s *AuthService) EnsureAdmin(ctx context.Context) error {
	count, err := s.users.CountAdmins(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := hashPassword(s.cfg.InitAdminPass)
	if err != nil {
		return err
	}
	return s.users.Create(ctx, &model.AppUser{
		Username: s.cfg.InitAdminUser, DisplayName: "系统管理员", PasswordHash: hash,
		Role: model.UserRoleAdmin, Enabled: true,
	})
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, clientIP, userAgent string) (*dto.LoginResponse, error) {
	user, err := s.users.GetByUsername(ctx, strings.TrimSpace(req.Username))
	if err != nil {
		_ = s.audit(ctx, nil, req.Username, false, "用户名或密码错误", clientIP, userAgent)
		return nil, fmt.Errorf("用户名或密码错误")
	}
	if !user.Enabled {
		_ = s.audit(ctx, &user.ID, user.Username, false, "用户已禁用", clientIP, userAgent)
		return nil, fmt.Errorf("用户已禁用")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		_ = s.audit(ctx, &user.ID, user.Username, false, "用户名或密码错误", clientIP, userAgent)
		return nil, fmt.Errorf("用户名或密码错误")
	}
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, &user.ID, user.Username, true, "", clientIP, userAgent)
	token, err := s.Sign(Claims{UserID: user.ID, Username: user.Username, Role: user.Role, Exp: now.Add(time.Duration(s.cfg.JWTExpireHours) * time.Hour).Unix()})
	if err != nil {
		return nil, err
	}
	return &dto.LoginResponse{AccessToken: token, User: userToCurrent(*user)}, nil
}

func (s *AuthService) Me(ctx context.Context, userID uint64) (*dto.CurrentUser, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	current := userToCurrent(*user)
	return &current, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID uint64, req dto.ChangePasswordRequest) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)) != nil {
		return fmt.Errorf("原密码错误")
	}
	if len(req.NewPassword) < 8 {
		return fmt.Errorf("新密码长度至少 8 位")
	}
	hash, err := hashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	now := time.Now()
	user.PasswordHash = hash
	user.PasswordUpdatedAt = &now
	return s.users.Update(ctx, user)
}

func (s *AuthService) ListUsers(ctx context.Context) ([]model.AppUser, error) {
	return s.users.List(ctx)
}

func (s *AuthService) CreateUser(ctx context.Context, req dto.SaveUserRequest, createdBy uint64) (*model.AppUser, error) {
	if req.Password == "" {
		return nil, fmt.Errorf("password is required")
	}
	role := chooseUserRole(req.Role)
	hash, err := hashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	user := &model.AppUser{
		Username: strings.TrimSpace(req.Username), DisplayName: req.DisplayName, PasswordHash: hash,
		Role: role, Enabled: enabled, CreatedBy: createdBy,
	}
	if user.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) UpdateUser(ctx context.Context, id uint64, req dto.SaveUserRequest) (*model.AppUser, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.DisplayName != "" {
		user.DisplayName = req.DisplayName
	}
	if req.Role != "" {
		user.Role = chooseUserRole(req.Role)
	}
	if req.Enabled != nil {
		user.Enabled = *req.Enabled
	}
	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, id uint64, password string) error {
	if len(password) < 8 {
		return fmt.Errorf("密码长度至少 8 位")
	}
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	now := time.Now()
	user.PasswordHash = hash
	user.PasswordUpdatedAt = &now
	return s.users.Update(ctx, user)
}

func (s *AuthService) Sign(claims Claims) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signingInput := header + "." + payload
	sig := hmacSHA256(signingInput, s.cfg.JWTSecret)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func (s *AuthService) Verify(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token")
	}
	expected := base64.RawURLEncoding.EncodeToString(hmacSHA256(parts[0]+"."+parts[1], s.cfg.JWTSecret))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, fmt.Errorf("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	if claims.Exp < time.Now().Unix() {
		return nil, fmt.Errorf("token expired")
	}
	if claims.UserID == 0 || claims.Username == "" {
		return nil, fmt.Errorf("invalid token claims")
	}
	return &claims, nil
}

func (s *AuthService) audit(ctx context.Context, userID *uint64, username string, success bool, reason, clientIP, userAgent string) error {
	return s.users.AuditLogin(ctx, &model.LoginAudit{UserID: userID, Username: username, Success: success, FailureReason: reason, ClientIP: clientIP, UserAgent: userAgent})
}

func hashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", fmt.Errorf("密码长度至少 8 位")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func hmacSHA256(input, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return mac.Sum(nil)
}

func chooseUserRole(role string) string {
	if role == model.UserRoleAdmin {
		return model.UserRoleAdmin
	}
	return model.UserRoleUser
}

func userToCurrent(user model.AppUser) dto.CurrentUser {
	return dto.CurrentUser{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Role: user.Role, Enabled: user.Enabled}
}

func IsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
