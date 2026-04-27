package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func userIDFromContext(ctx context.Context) (uint, error) {
	if ctx == nil {
		return 0, fmt.Errorf("未认证")
	}

	value := ctx.Value("userId")
	switch v := value.(type) {
	case json.Number:
		id, err := v.Int64()
		if err != nil || id <= 0 {
			return 0, fmt.Errorf("登录用户无效")
		}
		return uint(id), nil
	case float64:
		if v <= 0 {
			return 0, fmt.Errorf("登录用户无效")
		}
		return uint(v), nil
	case int:
		if v <= 0 {
			return 0, fmt.Errorf("登录用户无效")
		}
		return uint(v), nil
	case int64:
		if v <= 0 {
			return 0, fmt.Errorf("登录用户无效")
		}
		return uint(v), nil
	case uint:
		if v == 0 {
			return 0, fmt.Errorf("登录用户无效")
		}
		return v, nil
	case string:
		id, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
		if err != nil || id == 0 {
			return 0, fmt.Errorf("登录用户无效")
		}
		return uint(id), nil
	default:
		return 0, fmt.Errorf("未认证")
	}
}

func hasRole(roles []string, target string) bool {
	for _, role := range roles {
		if strings.TrimSpace(role) == target {
			return true
		}
	}
	return false
}
