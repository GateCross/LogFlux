package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils/logger"
	"logflux/internal/xerr"

	"logflux/common/password"
	usermodel "logflux/model/user"

	"github.com/golang-jwt/jwt/v4"
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

	if err := password.Verify(user.Password, req.Password); err != nil {
		return nil, xerr.NewBusinessErrorWith("用户名或密码错误")
	}
	if user.Status == 0 {
		return nil, xerr.NewBusinessErrorWith("用户已被禁用")
	}

	accessToken, refreshToken, err := buildTokenPair(int64(user.ID), s.svcCtx.Config.Auth.AccessSecret, s.svcCtx.Config.Auth.AccessExpire)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "生成登录令牌失败", err)
	}
	return &types.LoginResp{Token: accessToken, RefreshToken: refreshToken}, nil
}

// GetSetupStatus 判断系统是否已完成初始化（是否已有账号）。
func (s *AuthService) GetSetupStatus() (*types.SetupStatusResp, error) {
	count, err := s.svcCtx.UserModel.Count(s.ctx)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "检查初始化状态失败", err)
	}
	return &types.SetupStatusResp{Initialized: count > 0}, nil
}

// Setup 创建首个管理员账号并直接签发令牌。
// 仅当系统中没有任何账号时允许调用，避免初始化接口被抢注。
func (s *AuthService) Setup(req *types.SetupReq) (*types.LoginResp, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return nil, xerr.NewBusinessErrorWith("请输入用户名")
	}
	if err := password.Validate(req.Password); err != nil {
		return nil, xerr.NewBusinessErrorWith(err.Error())
	}

	count, err := s.svcCtx.UserModel.Count(s.ctx)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "检查初始化状态失败", err)
	}
	if count > 0 {
		return nil, xerr.NewBusinessErrorWith("系统已初始化，请直接登录")
	}

	hashed, err := password.Hash(req.Password)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "密码加密失败", err)
	}

	user := &usermodel.User{
		Username: username,
		Password: hashed,
	}
	// 并发下可能有另一个首次初始化请求同时通过上面的检查，
	// 依赖 users.username 唯一索引兜底：只有一个请求能插入成功。
	if err := s.svcCtx.UserModel.Create(s.ctx, user); err != nil {
		nextCount, countErr := s.svcCtx.UserModel.Count(s.ctx)
		if countErr == nil && nextCount > 0 {
			return nil, xerr.NewBusinessErrorWith("系统已初始化，请直接登录")
		}
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "创建账号失败", err)
	}

	s.Logger.Infof("首个管理员账号已创建: username=%s", username)
	return s.issueTokens(user)
}

// issueTokens 为指定用户签发访问令牌与刷新令牌。
func (s *AuthService) issueTokens(user *usermodel.User) (*types.LoginResp, error) {
	accessToken, refreshToken, err := buildTokenPair(
		int64(user.ID),
		s.svcCtx.Config.Auth.AccessSecret,
		s.svcCtx.Config.Auth.AccessExpire,
	)
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

	accessToken, refreshToken, err := buildTokenPair(int64(user.ID), secret, s.svcCtx.Config.Auth.AccessExpire)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "刷新令牌失败", err)
	}
	return &types.LoginResp{Token: accessToken, RefreshToken: refreshToken}, nil
}

// buildTokenPair 签发访问令牌与刷新令牌；删除 RBAC 后令牌不再携带角色。
func buildTokenPair(userID int64, secret string, accessExpire int64) (string, string, error) {
	accessToken, err := buildToken(userID, secret, accessExpire, accessTokenType)
	if err != nil {
		return "", "", err
	}

	refreshExpire := accessExpire * defaultRefreshExpireFactor
	if refreshExpire < minRefreshExpireSeconds {
		refreshExpire = minRefreshExpireSeconds
	}

	refreshToken, err := buildToken(userID, secret, refreshExpire, refreshTokenType)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func buildToken(userID int64, secret string, expireSeconds int64, tokenType string) (string, error) {
	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"exp":       now + expireSeconds,
		"iat":       now,
		"userId":    userID,
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
