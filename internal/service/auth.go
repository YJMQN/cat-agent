package service

import (
	"errors"
	"time"

	"cat-agent/internal/config"
	"cat-agent/internal/domain"
	"cat-agent/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService 认证服务
type AuthService struct {
	repo *repository.Repository
	cfg  *config.Config
}

// NewAuthService 创建认证服务
func NewAuthService(repo *repository.Repository, cfg *config.Config) *AuthService {
	return &AuthService{repo: repo, cfg: cfg}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string      `json:"token"`
	ExpiresAt int64       `json:"expires_at"`
	User      domain.User `json:"user"`
}

// Login 用户登录
func (s *AuthService) Login(req *LoginRequest) (*LoginResponse, error) {
	user, err := s.repo.User.GetByUsername(req.Username)
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	expiresAt := time.Now().Add(time.Duration(s.cfg.JWTExpire) * time.Hour)
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, errors.New("生成令牌失败")
	}

	return &LoginResponse{
		Token:     tokenStr,
		ExpiresAt: expiresAt.Unix(),
		User:      *user,
	}, nil
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6"`
}

// Register 用户注册
func (s *AuthService) Register(req *RegisterRequest) (*domain.User, error) {
	// 检查用户名是否已存在
	if _, err := s.repo.User.GetByUsername(req.Username); err == nil {
		return nil, errors.New("用户名已存在")
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("密码加密失败")
	}

	user := &domain.User{
		Username: req.Username,
		Password: string(hashedPwd),
		Role:     "user",
	}

	if err := s.repo.User.Create(user); err != nil {
		return nil, errors.New("创建用户失败")
	}

	return user, nil
}

// ParseToken 解析JWT令牌
func (s *AuthService) ParseToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("无效的签名方法")
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("无效的令牌")
	}

	return claims, nil
}
