package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	accessTokenType            = "access"
	refreshTokenType           = "refresh"
	defaultRefreshExpireFactor = int64(7)
	minRefreshExpireSeconds    = int64(24 * 60 * 60)
)

func buildTokenPair(userId int64, roles []string, secret string, accessExpire int64) (string, string, error) {
	role := "user"
	if len(roles) > 0 {
		role = roles[0]
	}

	accessToken, err := buildToken(userId, role, secret, accessExpire, accessTokenType)
	if err != nil {
		return "", "", err
	}

	refreshExpire := accessExpire * defaultRefreshExpireFactor
	if refreshExpire < minRefreshExpireSeconds {
		refreshExpire = minRefreshExpireSeconds
	}

	refreshToken, err := buildToken(userId, role, secret, refreshExpire, refreshTokenType)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func buildToken(userId int64, role string, secret string, expireSeconds int64, tokenType string) (string, error) {
	now := time.Now().Unix()
	claims := make(jwt.MapClaims)
	claims["exp"] = now + expireSeconds
	claims["iat"] = now
	claims["userId"] = userId
	claims["role"] = role
	claims["tokenType"] = tokenType

	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = claims

	return token.SignedString([]byte(secret))
}

func parseTokenClaims(tokenString string, secret string) (jwt.MapClaims, error) {
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func getTokenType(claims jwt.MapClaims) string {
	if tokenType, ok := claims["tokenType"].(string); ok {
		return tokenType
	}

	return ""
}

func getUserIdFromClaims(claims jwt.MapClaims) (int64, error) {
	rawUserId, ok := claims["userId"]
	if !ok {
		return 0, errors.New("missing userId")
	}

	switch userId := rawUserId.(type) {
	case float64:
		return int64(userId), nil
	case int64:
		return userId, nil
	case int:
		return int64(userId), nil
	case string:
		parsed, err := strconv.ParseInt(userId, 10, 64)
		if err != nil {
			return 0, errors.New("invalid userId")
		}

		return parsed, nil
	default:
		return 0, errors.New("invalid userId")
	}
}
