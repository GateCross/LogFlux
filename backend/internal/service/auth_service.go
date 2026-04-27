package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"logflux/common/cryptx"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils/logger"
	"logflux/internal/xerr"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	accessTokenType            = "access"
	refreshTokenType           = "refresh"
	defaultRefreshExpireFactor = int64(7)
	minRefreshExpireSeconds    = int64(24 * 60 * 60)
)

// AuthService 负责登录、令牌刷新等认证业务。
type AuthService struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewAuthService 创建认证服务。
func NewAuthService(ctx context.Context, svcCtx *svc.ServiceContext) *AuthService {
	return &AuthService{
		Logger: logger.New(logger.ModuleSystem).WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Login 校验用户密码并签发访问令牌和刷新令牌。
func (s *AuthService) Login(req *types.LoginReq) (*types.LoginResp, error) {
	user, err := s.svcCtx.UserModel.FindByUsername(s.ctx, req.Username, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, xerr.NewBusinessErrorWith("用户不存在")
		}
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询用户失败", err)
	}

	password, err := cryptx.Decrypt(req.Password, s.svcCtx.Config.Auth.AESKey)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith("用户名或密码错误")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, xerr.NewBusinessErrorWith("用户名或密码错误")
	}
	if user.Status == 0 {
		return nil, xerr.NewBusinessErrorWith("用户已被禁用")
	}

	accessToken, refreshToken, err := buildTokenPair(int64(user.ID), user.Roles, s.svcCtx.Config.Auth.AccessSecret, s.svcCtx.Config.Auth.AccessExpire)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "生成登录令牌失败", err)
	}
	return &types.LoginResp{Token: accessToken, RefreshToken: refreshToken}, nil
}

// RefreshToken 校验刷新令牌并签发新的令牌对。
func (s *AuthService) RefreshToken(req *types.RefreshTokenReq) (*types.LoginResp, error) {
	if req.RefreshToken == "" {
		return nil, xerr.NewCodeError(3000, "刷新令牌不能为空")
	}

	secret := s.svcCtx.Config.Auth.AccessSecret
	claims, err := parseTokenClaims(req.RefreshToken, secret)
	if err != nil {
		return nil, xerr.NewCodeError(3000, "刷新令牌无效或已过期")
	}
	if getTokenType(claims) != refreshTokenType {
		return nil, xerr.NewCodeError(3000, "刷新令牌类型错误")
	}

	userID, err := getUserIDFromClaims(claims)
	if err != nil {
		return nil, xerr.NewCodeError(3000, "刷新令牌无效")
	}
	user, err := s.svcCtx.UserModel.FindByID(s.ctx, uint(userID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, xerr.NewCodeError(3000, "用户不存在")
		}
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询用户失败", err)
	}
	if user.Status == 0 {
		return nil, xerr.NewCodeError(3000, "用户已被禁用")
	}

	accessToken, refreshToken, err := buildTokenPair(int64(user.ID), user.Roles, secret, s.svcCtx.Config.Auth.AccessExpire)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "刷新令牌失败", err)
	}
	return &types.LoginResp{Token: accessToken, RefreshToken: refreshToken}, nil
}

func buildTokenPair(userID int64, roles []string, secret string, accessExpire int64) (string, string, error) {
	role := "user"
	if len(roles) > 0 {
		role = roles[0]
	}

	accessToken, err := buildToken(userID, role, secret, accessExpire, accessTokenType)
	if err != nil {
		return "", "", err
	}

	refreshExpire := accessExpire * defaultRefreshExpireFactor
	if refreshExpire < minRefreshExpireSeconds {
		refreshExpire = minRefreshExpireSeconds
	}

	refreshToken, err := buildToken(userID, role, secret, refreshExpire, refreshTokenType)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func buildToken(userID int64, role string, secret string, expireSeconds int64, tokenType string) (string, error) {
	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"exp":       now + expireSeconds,
		"iat":       now,
		"userId":    userID,
		"role":      role,
		"tokenType": tokenType,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func parseTokenClaims(tokenString string, secret string) (jwt.MapClaims, error) {
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("令牌签名算法不支持: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return nil, fmt.Errorf("令牌无效")
	}
	return claims, nil
}

func getTokenType(claims jwt.MapClaims) string {
	if tokenType, ok := claims["tokenType"].(string); ok {
		return tokenType
	}
	return ""
}

func getUserIDFromClaims(claims jwt.MapClaims) (int64, error) {
	rawUserID, ok := claims["userId"]
	if !ok {
		return 0, fmt.Errorf("令牌缺少用户")
	}

	switch userID := rawUserID.(type) {
	case float64:
		return int64(userID), nil
	case int64:
		return userID, nil
	case int:
		return int64(userID), nil
	case string:
		return strconv.ParseInt(userID, 10, 64)
	default:
		return 0, fmt.Errorf("令牌用户无效")
	}
}
