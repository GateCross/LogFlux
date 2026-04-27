package caddy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type AddWafSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddWafSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddWafSourceLogic {
	return &AddWafSourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddWafSourceLogic) AddWafSource(req *types.WafSourceReq) (resp *types.BaseResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("源名称不能为空")
	}

	kind := normalizeWafKind(req.Kind)
	if err := validateWafKind(kind); err != nil {
		return nil, err
	}
	if kind == wafKindCorazaEngine {
		return nil, fmt.Errorf("Coraza 引擎更新源无需手工配置，请直接使用引擎版本检查")
	}

	mode := normalizeWafMode(req.Mode)
	if err := validateWafMode(mode); err != nil {
		return nil, err
	}

	authType := normalizeWafAuthType(req.AuthType)
	if err := validateWafAuthType(authType); err != nil {
		return nil, err
	}

	sourceURL := strings.TrimSpace(req.Url)
	if mode == wafModeRemote && sourceURL == "" {
		return nil, fmt.Errorf("远程源 URL 不能为空")
	}

	meta, err := parseMetaJSON(req.Meta)
	if err != nil {
		return nil, err
	}

	var existing model.WafSource
	if err := helper.svcCtx.DB.WithContext(helper.ctx).Where("name = ?", name).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("源名称已存在: %s", name)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查源名称失败: %w", err)
	}

	source := &model.WafSource{
		Name:         name,
		Kind:         kind,
		Mode:         mode,
		URL:          sourceURL,
		ChecksumURL:  strings.TrimSpace(req.ChecksumUrl),
		ProxyURL:     strings.TrimSpace(req.ProxyUrl),
		AuthType:     authType,
		AuthSecret:   strings.TrimSpace(req.AuthSecret),
		Schedule:     strings.TrimSpace(req.Schedule),
		Enabled:      true,
		AutoCheck:    true,
		AutoDownload: true,
		AutoActivate: false,
		Meta:         meta,
	}
	if kind == wafKindCorazaEngine {
		source.AutoActivate = false
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
	if kind == wafKindCorazaEngine {
		source.AutoActivate = false
	}

	if err := helper.svcCtx.DB.WithContext(helper.ctx).Create(source).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			return nil, fmt.Errorf("源名称已存在: %s", name)
		}
		return nil, fmt.Errorf("创建源失败: %w", err)
	}
	if helper.svcCtx.WafScheduler != nil {
		if reloadErr := helper.svcCtx.WafScheduler.ReloadSource(source.ID); reloadErr != nil {
			l.Logger.Errorf("重载 WAF 调度源失败: sourceID=%d err=%v", source.ID, reloadErr)
		}
	}

	return &types.BaseResp{Code: 200, Msg: "成功"}, nil
}
