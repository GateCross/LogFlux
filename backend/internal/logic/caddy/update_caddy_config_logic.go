package caddy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCaddyConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCaddyConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCaddyConfigLogic {
	return &UpdateCaddyConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCaddyConfigLogic) UpdateCaddyConfig(req *types.CaddyConfigUpdateReq) (resp *types.BaseResp, err error) {
	var server model.CaddyServer
	if err := l.svcCtx.DB.First(&server, req.ServerId).Error; err != nil {
		return nil, fmt.Errorf("server not found")
	}

	// 1. Save Caddyfile to Database (Source of Truth)
	server.Config = req.Config
	if req.Modules != "" {
		server.Modules = req.Modules
	}
	if err := l.svcCtx.DB.Save(&server).Error; err != nil {
		l.Logger.Errorf("Failed to save config to DB: %v", err)
		return nil, fmt.Errorf("failed to save config to database")
	}

	// 2. Push to Caddy API
	// API currently expects JSON by default, but we want to send Caddyfile.
	// We use /load endpoint with Content-Type: text/caddyfile
	// Caddy will compile it to JSON on the fly.

	l.Logger.Infof("Pushing Caddyfile to server %s (ID: %d)", server.Name, server.ID)

	url := fmt.Sprintf("%s/load", server.Url)
	reqBody := bytes.NewBufferString(req.Config)

	httpReq, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	// Critical: Tell Caddy this is a Caddyfile, not JSON
	httpReq.Header.Set("Content-Type", "text/caddyfile")

	// If remote auth is needed (not standard in default Caddy but possible via plugins or reverse proxy)
	if server.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+server.Token)
	}

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != 200 {
		body, _ := io.ReadAll(httpResp.Body)
		l.Logger.Errorf("Caddy API Error: %d - %s", httpResp.StatusCode, string(body))
		return nil, fmt.Errorf("caddy api error: %s", string(body))
	}

	// 3. 同步日志路径 (自动发现)
	// 更新成功后，我们从 Caddy 获取编译后的完整 JSON 配置
	// 这样可以解析出所有实际的日志文件路径（包括那些使用了变量的动态路径）
	go func() {
		// 等待 Caddy 加载完成
		// 简单的延时，或者直接查询
		syncUrl := fmt.Sprintf("%s/config/", server.Url)
		syncReq, _ := http.NewRequest("GET", syncUrl, nil)
		if server.Token != "" {
			syncReq.Header.Set("Authorization", "Bearer "+server.Token)
		}

		syncClient := &http.Client{}
		syncResp, err := syncClient.Do(syncReq)
		if err != nil {
			l.Logger.Errorf("同步日志配置失败: %v", err)
			return
		}
		defer syncResp.Body.Close()

		if syncResp.StatusCode == 200 {
			bodyBytes, _ := io.ReadAll(syncResp.Body)
			configStr := string(bodyBytes)

			// 使用正则提取所有的 "filename": "/path/..."
			// Caddy JSON 日志配置通常包含 "output": "file", "filename": "..."
			re := regexp.MustCompile(`"filename"\s*:\s*"([^"]+)"`)
			matches := re.FindAllStringSubmatch(configStr, -1)

			for _, match := range matches {
				if len(match) > 1 {
					path := match[1]
					l.Logger.Infof("发现日志文件: %s", path)

					// 检查数据库是否存在
					var count int64
					l.svcCtx.DB.Model(&model.LogSource{}).Where("path = ?", path).Count(&count)
					if count == 0 {
						// 创建新的日志源
						newSource := model.LogSource{
							Name:    fmt.Sprintf("Caddy Auto: %s", path),
							Path:    path,
							Type:    "caddy",
							Enabled: true,
						}
						if err := l.svcCtx.DB.Create(&newSource).Error; err == nil {
							l.Logger.Infof("自动添加日志源: %s", path)
							// 立即启动监控
							l.svcCtx.Ingestor.Start(path)
						}
					} else {
						// 确保它是启动状态
						l.svcCtx.Ingestor.Start(path)
					}
				}
			}
		}
	}()

	l.Logger.Info("Caddy config updated successfully")
	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}
