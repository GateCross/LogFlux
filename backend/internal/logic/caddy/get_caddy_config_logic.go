package caddy

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCaddyConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCaddyConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaddyConfigLogic {
	return &GetCaddyConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCaddyConfigLogic) GetCaddyConfig(req *types.CaddyConfigReq) (resp *types.CaddyConfigResp, err error) {
	var server model.CaddyServer
	if err := l.svcCtx.DB.First(&server, req.ServerId).Error; err != nil {
		return nil, fmt.Errorf("server not found")
	}

	url := fmt.Sprintf("%s/config/", server.Url)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if server.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+server.Token)
	}

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode != 200 {
		return nil, fmt.Errorf("caddy api error: %s", string(body))
	}

	return &types.CaddyConfigResp{
		Code:   200,
		Msg:    "success",
		Config: string(body),
	}, nil
}
