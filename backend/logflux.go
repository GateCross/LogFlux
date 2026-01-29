package main

import (
	"flag"
	"fmt"
	"net/http"

	"logflux/common/result"
	"logflux/internal/config"
	"logflux/internal/handler"
	"logflux/internal/middleware"
	"logflux/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/config.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf, rest.WithUnauthorizedCallback(func(w http.ResponseWriter, r *http.Request, err error) {
		httpx.OkJson(w, result.ResponseBean{
			Code: 401,
			Msg:  "Unauthorized",
			Data: nil,
		})
	}))
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// Global Response Middleware
	server.Use(middleware.ResponseMiddleware)

	// Global Error Handler (still needed for business errors)

	httpx.SetErrorHandler(func(err error) (int, any) {
		errCode := 500
		errMsg := err.Error()

		if ce, ok := err.(*result.CodeError); ok {
			errCode = ce.Code
			errMsg = ce.Msg
		}

		return 200, result.ResponseBean{ // Always return 200 status for business logic errors if that's what frontend expects, or use errCode if strictly HTTP status
			Code: errCode,
			Msg:  errMsg,
			Data: nil,
		}
	})

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
