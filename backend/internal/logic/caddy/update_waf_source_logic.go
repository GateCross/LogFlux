package caddy

import (
	"context"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateWafSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateWafSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateWafSourceLogic {
	return &UpdateWafSourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateWafSourceLogic) UpdateWafSource(req *types.WafSourceUpdateReq) (resp *types.BaseResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)

	var source model.WafSource
	if err := helper.svcCtx.DB.First(&source, req.ID).Error; err != nil {
		return nil, fmt.Errorf("source not found")
	}

	if name := strings.TrimSpace(req.Name); name != "" {
		source.Name = name
	}

	if strings.TrimSpace(req.Kind) != "" {
		kind := normalizeWafKind(req.Kind)
		if err := validateWafKind(kind); err != nil {
			return nil, err
		}
		source.Kind = kind
	}

	if strings.TrimSpace(req.Mode) != "" {
		mode := normalizeWafMode(req.Mode)
		if err := validateWafMode(mode); err != nil {
			return nil, err
		}
		source.Mode = mode
	}

	if strings.TrimSpace(req.AuthType) != "" {
		authType := normalizeWafAuthType(req.AuthType)
		if err := validateWafAuthType(authType); err != nil {
			return nil, err
		}
		source.AuthType = authType
	}

	if strings.TrimSpace(req.Url) != "" {
		source.URL = strings.TrimSpace(req.Url)
	}
	if strings.TrimSpace(req.ChecksumUrl) != "" {
		source.ChecksumURL = strings.TrimSpace(req.ChecksumUrl)
	}
	source.ProxyURL = strings.TrimSpace(req.ProxyUrl)
	if strings.TrimSpace(req.AuthSecret) != "" {
		source.AuthSecret = strings.TrimSpace(req.AuthSecret)
	}
	if strings.TrimSpace(req.Schedule) != "" {
		source.Schedule = strings.TrimSpace(req.Schedule)
	}
	if strings.TrimSpace(req.Meta) != "" {
		meta, err := parseMetaJSON(req.Meta)
		if err != nil {
			return nil, err
		}
		source.Meta = meta
	}

	if helper.hasSourceBoolField("enabled") {
		source.Enabled = req.Enabled
	}
	if helper.hasSourceBoolField("autoCheck") {
		source.AutoCheck = req.AutoCheck
	}
	if helper.hasSourceBoolField("autoDownload") {
		source.AutoDownload = req.AutoDownload
	}
	if helper.hasSourceBoolField("autoActivate") {
		source.AutoActivate = req.AutoActivate
	}
	if source.Kind == wafKindCorazaEngine {
		source.AutoActivate = false
	}

	if source.Mode == wafModeRemote && strings.TrimSpace(source.URL) == "" {
		return nil, fmt.Errorf("url is required for remote source")
	}

	if err := helper.svcCtx.DB.Save(&source).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			return nil, fmt.Errorf("source name already exists: %s", source.Name)
		}
		return nil, fmt.Errorf("update source failed: %w", err)
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
