package caddy

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckWafSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCheckWafSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckWafSourceLogic {
	return &CheckWafSourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CheckWafSourceLogic) CheckWafSource(req *types.WafSourceActionReq) (resp *types.BaseResp, err error) {
	helper := newWafLogicHelper(l.ctx, l.svcCtx, l.Logger)

	var source model.WafSource
	if err := helper.svcCtx.DB.First(&source, req.ID).Error; err != nil {
		return nil, fmt.Errorf("source not found")
	}

	job := helper.startJob(source.ID, 0, "check", "manual")

	if err := validateWafKind(source.Kind); err != nil {
		helper.updateSourceLastCheck(source.ID, "", err.Error())
		helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
		return nil, err
	}
	if err := validateWafMode(source.Mode); err != nil {
		helper.updateSourceLastCheck(source.ID, "", err.Error())
		helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
		return nil, err
	}
	if err := validateWafAuthType(source.AuthType); err != nil {
		helper.updateSourceLastCheck(source.ID, "", err.Error())
		helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
		return nil, err
	}

	if source.Mode == wafModeRemote {
		if strings.TrimSpace(source.URL) == "" {
			err = fmt.Errorf("url is required for remote source")
			helper.updateSourceLastCheck(source.ID, "", err.Error())
			helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
			return nil, err
		}
		parsedURL, parseErr := url.Parse(strings.TrimSpace(source.URL))
		if parseErr != nil {
			err = fmt.Errorf("invalid url: %w", parseErr)
			helper.updateSourceLastCheck(source.ID, "", err.Error())
			helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
			return nil, err
		}
		if parsedURL.Scheme != "https" {
			err = fmt.Errorf("only https url is allowed")
			helper.updateSourceLastCheck(source.ID, "", err.Error())
			helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
			return nil, err
		}

		if proxyValue := strings.TrimSpace(source.ProxyURL); proxyValue != "" {
			proxyURL, proxyErr := url.Parse(proxyValue)
			if proxyErr != nil {
				err = fmt.Errorf("invalid proxy url: %w", proxyErr)
				helper.updateSourceLastCheck(source.ID, "", err.Error())
				helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
				return nil, err
			}
			if proxyURL.Scheme != "http" && proxyURL.Scheme != "https" {
				err = fmt.Errorf("proxy url scheme must be http or https")
				helper.updateSourceLastCheck(source.ID, "", err.Error())
				helper.finishJob(job, wafJobStatusFailed, err.Error(), 0)
				return nil, err
			}
		}
	}

	helper.updateSourceLastCheck(source.ID, "", "")
	helper.finishJob(job, wafJobStatusSuccess, "check success", 0)
	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}
