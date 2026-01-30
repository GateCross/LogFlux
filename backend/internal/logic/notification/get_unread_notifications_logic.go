package notification

import (
	"context"
	"encoding/json"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUnreadNotificationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUnreadNotificationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUnreadNotificationsLogic {
	return &GetUnreadNotificationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUnreadNotificationsLogic) GetUnreadNotifications() (resp *types.LogListResp, err error) {
	// 1. 获取当前用户 ID
	userIdVal := l.ctx.Value("userId")
	var uid int64
	if jsonUid, ok := userIdVal.(json.Number); ok {
		if id, err := jsonUid.Int64(); err == nil {
			uid = id
		}
	} else if floatUid, ok := userIdVal.(float64); ok {
		uid = int64(floatUid)
	} else if intUid, ok := userIdVal.(int); ok {
		uid = int64(intUid)
	}

	// 2. 获取用户偏好设置 (minLevel)
	minLevel := "info" // 默认级别
	levelMap := map[string]int{
		"debug":    0,
		"info":     1,
		"warning":  2,
		"error":    3,
		"critical": 4,
	}

	if uid > 0 {
		var user model.User
		if err := l.svcCtx.DB.First(&user, uid).Error; err == nil && user.Preferences != nil {
			// 解析 JSONMap
			if levelObj, ok := user.Preferences["minLevel"]; ok {
				if levelStr, ok := levelObj.(string); ok {
					minLevel = strings.ToLower(levelStr)
				}
			}
		}
	}

	minLevelScore := levelMap[minLevel]

	// 3. 查询未读通知
	var logs []model.NotificationLog
	// 查找类型为 in_app 且 status=success 且 is_read=false 的日志
	err = l.svcCtx.DB.Table("notification_logs").
		Select("notification_logs.*").
		Joins("left join notification_channels on notification_channels.id = notification_logs.channel_id").
		Where("notification_channels.type = ?", "in_app").
		Where("notification_logs.status = ?", model.NotificationStatusSuccess).
		Where("notification_logs.is_read = ?", false).
		Order("notification_logs.created_at desc").
		Limit(50).
		Find(&logs).Error

	if err != nil {
		l.Logger.Errorf("Failed to get unread notifications: %v", err)
		return nil, err
	}

	// 4. 过滤和转换
	list := make([]types.LogItem, 0)
	for _, item := range logs {
		// 提取信息
		message := ""
		title := item.EventType
		level := "info"

		if item.EventData != nil {
			if content, ok := item.EventData["rendered_content"].(string); ok {
				message = content
			}
			if msg, ok := item.EventData["message"].(string); ok && message == "" {
				message = msg
			}
			if t, ok := item.EventData["title"].(string); ok {
				title = t
			}
			if l, ok := item.EventData["level"].(string); ok {
				level = l
			}
		}

		// **过滤逻辑**: 如果通知等级低于用户设置的最小等级，则跳过
		currentLevelScore := levelMap[strings.ToLower(level)]
		// 如果无法识别等级，默认为 info (1)
		if _, ok := levelMap[strings.ToLower(level)]; !ok {
			currentLevelScore = 1
		}

		if currentLevelScore < minLevelScore {
			continue
		}

		sentAt := ""
		if item.SentAt != nil {
			sentAt = item.SentAt.Format("2006-01-02 15:04:05")
		}

		ruleID := uint(0)
		if item.RuleID != nil {
			ruleID = *item.RuleID
		}

		logItem := types.LogItem{
			ID:         item.ID,
			EventID:    item.EventType,
			EventType:  item.EventType,
			Title:      title,
			Message:    message,
			Level:      level,
			ChannelID:  0, // Will set below
			RuleID:     ruleID,
			Status:     2, // success
			Error:      item.ErrorMessage,
			RetryCount: 0,
			SentAt:     sentAt,
			CreatedAt:  item.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		if item.ChannelID != nil {
			logItem.ChannelID = uint(*item.ChannelID)
		}
		list = append(list, logItem)
	}

	return &types.LogListResp{
		List:  list,
		Total: int64(len(list)), // Total 应该是过滤后的数量
	}, nil
}
